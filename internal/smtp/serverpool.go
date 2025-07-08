// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/adampresley/webframework/sanitizer"

	"github.com/mailslurper/mailslurper/v2/internal/mailslurper"
	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// ServerPool represents a pool of SMTP workers. This will manage how many workers may respond to SMTP client requests
// and allocation of those workers.
type ServerPool struct {
	logger *slog.Logger
	pool   chan *Worker
}

// JoinQueue adds a worker to the queue.
func (pool *ServerPool) JoinQueue(worker *Worker) {
	pool.pool <- worker
}

// NewServerPool creates a new server pool with a maximum number of SMTP workers. An array of workers is initialized
// with an ID and an initial state of SMTP_WORKER_IDLE.
func NewServerPool(maxWorkers int, xss sanitizer.IXSSServiceProvider, logger *slog.Logger) *ServerPool {
	emailValidationService := mailslurper.NewEmailValidationService()

	pool := &ServerPool{
		pool:   make(chan *Worker, maxWorkers),
		logger: logger,
	}

	for idx := range maxWorkers {
		pool.JoinQueue(NewWorker(
			idx+1,
			pool,
			emailValidationService,
			xss,
			logger.With("who", fmt.Sprintf("SMTP Worker %d", idx+1)),
		))
	}

	logger.Info("Worker pool configured", "workers", maxWorkers)

	return pool
}

// NextWorker retrieves the next available worker from the queue.
func (pool *ServerPool) NextWorker(
	connection net.Conn,
	receiver chan *model.MailItem,
	chStop chan struct{},
	connectionCloseChannel chan net.Conn,
) (*Worker, error) {
	select {
	case worker := <-pool.pool:
		worker.Prepare(
			connection,
			receiver,
			&Reader{
				Connection: connection,
				logger:     pool.logger.With("who", fmt.Sprintf("SMTP Reader %d", worker.WorkerID)),
				chStop:     chStop,
			},
			&Writer{
				Connection: connection,
				logger:     pool.logger.With("who", fmt.Sprintf("SMTP Writer %d", worker.WorkerID)),
			},
			chStop,
			connectionCloseChannel,
		)

		pool.logger.Info(fmt.Sprintf("Worker %d queued to handle connections from %s", worker.WorkerID, connection.RemoteAddr().String()))

		return worker, nil

	case <-time.After(time.Second * 2):
		return &Worker{}, NoWorkerAvailable()
	}
}
