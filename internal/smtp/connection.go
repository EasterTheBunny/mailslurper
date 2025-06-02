// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/pkg/errors"

	slurperio "github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// ConnectionManager is responsible for maintaining, closing, and cleaning client connections. For every connection
// there is a worker. After an idle timeout period the manager will forceably close a client connection.
type ConnectionManager struct {
	closeChannel    chan net.Conn
	config          *slurperio.Config
	connectionPool  ConnectionPool
	chStop          chan struct{}
	logger          *slog.Logger
	mailItemChannel chan *model.MailItem
	serverPool      *ServerPool
}

// NewConnectionManager creates a new struct.
func NewConnectionManager(
	logger *slog.Logger,
	config *slurperio.Config,
	chStop chan struct{},
	mailItemChannel chan *model.MailItem,
	serverPool *ServerPool,
) *ConnectionManager {
	closeChannel := make(chan net.Conn, 5)

	result := &ConnectionManager{
		closeChannel:    closeChannel,
		config:          config,
		connectionPool:  NewConnectionPool(),
		chStop:          chStop,
		logger:          logger,
		mailItemChannel: mailItemChannel,
		serverPool:      serverPool,
	}

	go func() {
		var err error

		for {
			select {
			case connection := <-closeChannel:
				err = result.Close(connection)
				if err != nil && err != io.EOF {
					logger.With("error", err).Error("Error closing connection")
				}

				logger.With("connection", connection.RemoteAddr().String()).Info("Connection closed")

				break
			case <-chStop:
				return
			}
		}
	}()

	return result
}

// Close will close a client connection. The state of the worker is only used for logging purposes.
func (m *ConnectionManager) Close(connection net.Conn) error {
	if m.connectionExistsInPool(connection) {
		if !m.isConnectionClosed(connection) {
			m.logger.Info(fmt.Sprintf("Closing connection %s", connection.RemoteAddr().String()))

			return m.connectionPool[connection.RemoteAddr().String()].Connection.Close()
		}

		return nil
	}

	return ConnectionNotExists(connection.RemoteAddr().String())
}

func (m *ConnectionManager) connectionExistsInPool(connection net.Conn) bool {
	if _, ok := m.connectionPool[connection.RemoteAddr().String()]; ok {
		return true
	}

	return false
}

func (m *ConnectionManager) isConnectionClosed(connection net.Conn) bool {
	var err error

	temp := []byte{}

	if err = connection.SetReadDeadline(time.Now()); err != nil {
		return true
	}

	if _, err = connection.Read(temp); err == io.EOF {
		return true
	}

	return false
}

// New attempts to track a new client connection. The SMTPListener will use this to track a client connection and its
// worker.
func (m *ConnectionManager) New(connection net.Conn) error {
	var err error
	var worker *Worker

	if m.connectionExistsInPool(connection) {
		return ConnectionExists(connection.RemoteAddr().String())
	}

	if worker, err = m.serverPool.NextWorker(connection, m.mailItemChannel, m.chStop, m.closeChannel); err != nil {
		connection.Close()
		m.logger.With("error", err).Error("Error getting next SMTP worker")

		return errors.Wrapf(err, "Error getting work in ConnectionManager")
	}

	m.connectionPool[connection.RemoteAddr().String()] = NewConnectionPoolItem(connection, worker)
	go m.connectionPool[connection.RemoteAddr().String()].Worker.Work()

	return nil
}

// ConnectionPool is a map of remote address to TCP connections and their workers.
type ConnectionPool map[string]*ConnectionPoolItem

// ConnectionPoolItem is a single item in the pool. This tracks a connection to its worker.
type ConnectionPoolItem struct {
	Connection net.Conn
	Worker     *Worker
}

// NewConnectionPool creates a new empty map.
func NewConnectionPool() ConnectionPool {
	return make(ConnectionPool)
}

// NewConnectionPoolItem create a new object.
func NewConnectionPoolItem(connection net.Conn, worker *Worker) *ConnectionPoolItem {
	return &ConnectionPoolItem{
		Connection: connection,
		Worker:     worker,
	}
}
