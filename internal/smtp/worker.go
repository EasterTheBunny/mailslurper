// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/adampresley/webframework/sanitizer"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"github.com/mailslurper/mailslurper/v2/internal/mailslurper"
	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// Worker is responsible for executing, parsing, and processing a single TCP connection's email.
type Worker struct {
	Connection             net.Conn
	EmailValidationService mailslurper.EmailValidationProvider
	Error                  error
	Reader                 *Reader
	Receiver               chan *model.MailItem
	State                  SMTPWorkerState
	WorkerID               int
	Writer                 *Writer
	XSSService             sanitizer.IXSSServiceProvider

	connectionCloseChannel chan net.Conn
	chStop                 chan struct{}
	pool                   *ServerPool
	logger                 *slog.Logger
}

type smtpCommand struct {
	Command     Command
	StreamInput string
}

// NewWorker creates a new SMTP worker. An SMTP worker is responsible for parsing and working with SMTP mail data.
func NewWorker(
	workerID int,
	pool *ServerPool,
	emailValidationService mailslurper.EmailValidationProvider,
	xssService sanitizer.IXSSServiceProvider,
	logger *slog.Logger,
) *Worker {
	return &Worker{
		EmailValidationService: emailValidationService,
		WorkerID:               workerID,
		State:                  SMTP_WORKER_IDLE,
		XSSService:             xssService,

		pool:   pool,
		logger: logger,
	}
}

// Prepare tells a worker about the TCP connection they will work with, the IO handlers, and sets their state.
func (w *Worker) Prepare(
	connection net.Conn,
	receiver chan *model.MailItem,
	reader *Reader,
	writer *Writer,
	chStop chan struct{},
	connectionCloseChannel chan net.Conn,
) {
	w.State = SMTP_WORKER_WORKING

	w.Connection = connection
	w.Receiver = receiver

	w.Reader = reader
	w.Writer = writer

	w.connectionCloseChannel = connectionCloseChannel
	w.chStop = chStop
}

func (w *Worker) rejoinWorkerQueue() {
	w.pool.JoinQueue(w)
}

// Work is the function called by the SmtpListener when a client request is received. This will start the process by
// responding to the client, start processing commands, and finally close the connection.
func (w *Worker) Work() {
	var err error

	w.Writer.SayHello()
	mailItem := model.NewEmptyMailItem(w.logger)

	quitChannel := make(chan bool, 2)
	quitCommandChannel := make(chan bool, 2)
	workerErrorChannel := make(chan error, 2)
	commandChannel := make(chan smtpCommand)
	commandDoneChannel := make(chan error)

	/*
	 * This goroutine is the command processor
	 */
	go func() {
		var streamInput string
		var command Command
		var err error
		var networkError net.Error
		var ok bool

		for {
			select {
			case <-quitCommandChannel:
				return

			default:
				if streamInput, err = w.Reader.Read(); err != nil {
					if networkError, ok = err.(net.Error); ok {
						if networkError.Timeout() {
							w.logger.With("connection", w.Connection.RemoteAddr().String()).Info("Connection inactivity timeout")

							quitCommandChannel <- true
							quitChannel <- true
							break
						}
					}

					workerErrorChannel <- err
					break
				}

				if command, err = GetCommandFromString(streamInput); err != nil {
					w.logger.With("input", streamInput).Error("Problem finding command from input", "error", err)
					workerErrorChannel <- errors.Wrapf(err, "Problem finding command from input %s", streamInput)
					break
				}

				if command == QUIT {
					quitCommandChannel <- true
					quitChannel <- true
					break
				}

				commandChannel <- smtpCommand{Command: command, StreamInput: streamInput}
				err = <-commandDoneChannel

				if err != nil {
					w.logger.Error("Error executing command", "error", err)
					quitCommandChannel <- true
				}
			}
		}
	}()

	for {
		select {
		case <-w.chStop:
			w.State = SMTP_WORKER_DONE
			w.Writer.SayGoodbye()
			w.connectionCloseChannel <- w.Connection
			break

		case <-quitChannel:
			w.logger.With("connection", w.Connection.RemoteAddr().String()).Info("QUIT command received")
			w.Writer.SayGoodbye()

			w.State = SMTP_WORKER_DONE
			w.connectionCloseChannel <- w.Connection
			w.rejoinWorkerQueue()

			break

		case workerError := <-workerErrorChannel:
			w.State = SMTP_WORKER_ERROR
			w.Error = workerError
			w.Writer.SayGoodbye()

			w.connectionCloseChannel <- w.Connection
			w.rejoinWorkerQueue()
			break

		case command := <-commandChannel:
			if command.Command == QUIT {
				quitChannel <- true
				continue
			}

			executor := w.getExecutorFromCommand(command.Command)
			command.StreamInput = strings.TrimSpace(command.StreamInput)

			if err = executor.Process(command.StreamInput, mailItem); err != nil {
				w.logger.With("command", command.Command.String(), "input", command.StreamInput).Error("Problem executing command", "error", err)
				workerErrorChannel <- errors.Wrapf(err, "Problem executing command %s (stream input == '%s')", command.Command.String(), command.StreamInput)

				commandDoneChannel <- err
				continue
			}

			if command.Command == DATA {
				copy := model.NewEmptyMailItem(w.logger)
				copier.Copy(copy, mailItem)
				w.Receiver <- copy

				mailItem = model.NewEmptyMailItem(w.logger)
			}

			commandDoneChannel <- nil
		}
	}
}

func (w *Worker) getExecutorFromCommand(command Command) mailslurper.ICommandExecutor {
	switch command {
	case MAIL:
		return NewMailCommandExecutor(
			w.logger.With("who", "MAIL Command Executor"),
			w.Reader,
			w.Writer,
			w.EmailValidationService,
			w.XSSService,
		)
	case RCPT:
		return NewRcptCommandExecutor(
			w.logger.With("who", "RCPT TO Command Executor"),
			w.Reader,
			w.Writer,
			w.EmailValidationService,
			w.XSSService,
		)
	case DATA:
		return NewDataCommandExecutor(
			w.logger.With("who", "DATA Command Executor"),
			w.Reader,
			w.Writer,
			w.EmailValidationService,
			w.XSSService,
		)
	case RSET:
		return NewResetCommandExecutor(
			w.logger.With("who", "RSET Command Executor"),
			w.Writer,
		)
	case NOOP:
		return NewNoopCommandExecutor(
			w.logger.With("who", "NOOP Command Executor"),
			w.Writer,
		)
	default:
		return NewHelloCommandExecutor(
			w.logger.With("who", "HELO Command Executor"),
			w.Reader,
			w.Writer,
		)
	}
}

// TimeoutHasExpired determines if the time elapsed since a start time has exceeded the command timeout.
func (w *Worker) TimeoutHasExpired(startTime time.Time) bool {
	return int(time.Since(startTime).Seconds()) > COMMAND_TIMEOUT_SECONDS
}
