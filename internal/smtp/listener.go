// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"

	"github.com/pkg/errors"

	slurperio "github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/mailslurper"
	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// Listener sets up a server that listens on a TCP socket for connections. When a connection is received a worker is
// created to handle processing the mail on this connection.
type Listener struct {
	certificate         tls.Certificate
	config              slurperio.ListenConfig
	connectionManager   mailslurper.IConnectionManager
	killListenerChannel chan bool
	killRecieverChannel chan bool
	listener            net.Listener
	logger              *slog.Logger
	mailItemChannel     chan *model.MailItem
	receivers           []mailslurper.IMailItemReceiver
	serverPool          *ServerPool

	// internal service state
	chClose chan struct{}
	running atomic.Bool
}

// NewListener creates an Listener struct.
func NewListener(
	logger *slog.Logger,
	config slurperio.ListenConfig,
	mailItemChannel chan *model.MailItem,
	serverPool *ServerPool,
	receivers []mailslurper.IMailItemReceiver,
	connectionManager mailslurper.IConnectionManager,
) (*Listener, error) {
	var err error

	result := &Listener{
		config:              config,
		connectionManager:   connectionManager,
		killListenerChannel: make(chan bool, 1),
		killRecieverChannel: make(chan bool, 1),
		logger:              logger,
		mailItemChannel:     mailItemChannel,
		receivers:           receivers,
		serverPool:          serverPool,
		chClose:             make(chan struct{}),
	}

	if config.CertFile != "" && config.KeyFile != "" {
		if result.certificate, err = tls.LoadX509KeyPair(config.CertFile, config.KeyFile); err != nil {
			return result, errors.Wrapf(err, "Error loading X509 certificate key pair while setting up SMTP listener")
		}
	}

	return result, nil
}

// ListenAndServe starts the process of handling SMTP client connections. The first order of business is to setup a
// channel for writing parsed mails, in the form of MailItemStruct variables, to our database. A goroutine is setup to
// listen on that channel and handles storage.
//
// Meanwhile this method will loop forever and wait for client connections (blocking). When a connection is recieved a
// goroutine is started to create a new MailItemStruct and parser and the parser process is started. If the parsing is
// successful the MailItemStruct is added to a channel. An receivers passed in will be listening on that channel and may
// do with the mail item as they wish.
//
// ListenAndServe always returns a non-nil error.
func (l *Listener) ListenAndServe() error {
	l.running.Store(true)
	if err := l.openListener(); err != nil {
		l.running.Store(false)

		return err
	}

	go l.startReceivers()

	// block and accept connections until listener is shutdown
	l.acceptConnections()

	return ErrServerClosed
}

func (l *Listener) Shutdown(_ context.Context) error {
	close(l.chClose)
	l.running.Store(false)

	return nil
}

func (l *Listener) Close() error {
	close(l.chClose)
	l.running.Store(false)

	return nil
}

func (l *Listener) openListener() error {
	var (
		tcpAddress *net.TCPAddr
		err        error
	)

	if l.config.IsSSL() {
		tlsConfig := &tls.Config{Certificates: []tls.Certificate{l.certificate}}

		if l.listener, err = tls.Listen("tcp", l.config.GetBindingAddress(), tlsConfig); err != nil {
			return errors.Wrapf(err, "Unable to start SMTP listener using TLS")
		}

		l.logger.Info(fmt.Sprintf("SMTP listener running on SSL %s", l.config.GetBindingAddress()))
	} else {
		if tcpAddress, err = net.ResolveTCPAddr("tcp", l.config.GetBindingAddress()); err != nil {
			return errors.Wrapf(err, "Error resolving address %s starting SMTP listener", l.config.GetBindingAddress())
		}

		if l.listener, err = net.ListenTCP("tcp", tcpAddress); err != nil {
			return errors.Wrapf(err, "Unable to start SMTP listener")
		}

		l.logger.Info(fmt.Sprintf("SMTP listener running on %s", l.config.GetBindingAddress()))
	}

	return nil
}

func (l *Listener) startReceivers() {
	l.logger.Info(fmt.Sprintf("%d receiver(s) listening", len(l.receivers)))

	for {
		select {
		case item := <-l.mailItemChannel:
			for _, r := range l.receivers {
				go r.Receive(item)
			}
		case <-l.chClose:
			l.logger.Info("Shutting down receiver channel...")

			break
		}
	}
}

func (l *Listener) acceptConnections() {
	for {
		select {
		case <-l.chClose:
			break
		default:
			connection, err := l.listener.Accept()
			if err != nil {
				l.logger.With("error", err).Error("Problem accepting SMTP requests")
				close(l.chClose)
				l.running.Store(false)

				break
			}

			if err = l.connectionManager.New(connection); err != nil {
				l.logger.With("error", err).Error(fmt.Sprintf("Error adding connection '%s' to connection manager", connection.RemoteAddr().String()))
				connection.Close()
			}
		}
	}
}

// Deprecated:
//
// Start establishes a listening connection to a socket on an address.
func (l *Listener) Start() error {
	return l.openListener()
}

// Deprecated:
//
// Dispatch starts the process of handling SMTP client connections. The first order of business is to setup a channel
// for writing parsed mails, in the form of MailItemStruct variables, to our database. A goroutine is setup to listen on
// that channel and handles storage.
//
// Meanwhile this method will loop forever and wait for client connections (blocking). When a connection is recieved a
// goroutine is started to create a new MailItemStruct and parser and the parser process is started. If the parsing is
// successful the MailItemStruct is added to a channel. An receivers passed in will be listening on that channel and may
// do with the mail item as they wish.
func (l *Listener) Dispatch(ctx context.Context) {
	/*
	 * Setup our receivers. These guys are basically subscribers to
	 * the MailItem channel.
	 */
	go l.startReceivers()

	/*
	 * Now start accepting connections for SMTP. Add them to the connection manager
	 */
	go l.acceptConnections()
}
