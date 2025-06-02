// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"
)

// Writer is a simple object for writing commands and responses to a client connected on a TCP socket.
type Writer struct {
	Connection net.Conn

	logger *slog.Logger
}

// SayGoodbye tells a client that we are done communicating. This sends a 221 response. It returns true/false for
// success and a string with any response.
func (w *Writer) SayGoodbye() error {
	return w.SendResponse(SMTP_CLOSING_MESSAGE)
}

// SayHello sends a hello message to a new client. The SMTP protocol dictates that you must be polite. :)
func (w *Writer) SayHello() error {
	if err := w.SendResponse(SMTP_WELCOME_MESSAGE); err != nil {
		return err
	}

	w.logger.Info("Reading data from client connection...")

	return nil
}

// SendDataResponse is a function to send a DATA response message.
func (w *Writer) SendDataResponse() error {
	return w.SendResponse(SMTP_DATA_RESPONSE_MESSAGE)
}

// SendResponse sends a response to a client connection. It returns true/false for success and a string with any
// response.
func (w *Writer) SendResponse(response string) error {
	var err error

	if err = w.Connection.SetWriteDeadline(time.Now().Add(time.Second * 2)); err != nil {
		if !strings.Contains(err.Error(), "use of closed network connection") {
			w.logger.Error(fmt.Sprintf("Problem setting write deadline: %s", err.Error()))
		}
	}

	_, err = w.Connection.Write([]byte(string(response + SMTP_CRLF)))

	return err
}

// SendHELOResponse sends a HELO message to a client.
func (w *Writer) SendHELOResponse() error {
	return w.SendResponse(SMTP_HELLO_RESPONSE_MESSAGE)
}

// SendOkResponse sends an OK to a client.
func (w *Writer) SendOkResponse() error {
	return w.SendResponse(SMTP_OK_MESSAGE)
}
