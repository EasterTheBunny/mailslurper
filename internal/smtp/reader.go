// Copyright 2013-3014 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"
)

// Reader is a simple object for reading commands and responses from a connected TCP client.
type Reader struct {
	Connection net.Conn

	logger *slog.Logger
	chStop chan struct{}
}

// Read reads the raw data from the socket connection to our client. This will read on the socket until there is nothing
// left to read and an error is generated. This method blocks the socket for the number of milliseconds defined in
// CONN_TIMEOUT_MILLISECONDS. It then records what has been read in that time, then blocks again until there is nothing
// left on the socket to read. The final value is stored and returned as a string.
func (r *Reader) Read() (string, error) {
	var raw bytes.Buffer
	var bytesRead int
	var err error

	bytesRead = 1

	for bytesRead > 0 {
		select {
		case <-r.chStop:
			return "", nil

		default:
			if err = r.Connection.SetReadDeadline(time.Now().Add(time.Minute * CONNECTION_TIMEOUT_MINUTES)); err != nil {
				return raw.String(), nil
			}

			buffer := make([]byte, RECEIVE_BUFFER_LEN)
			bytesRead, err = r.Connection.Read(buffer)

			if err != nil {
				return raw.String(), err
			}

			if bytesRead > 0 {
				raw.WriteString(string(buffer[:bytesRead]))
				if strings.HasSuffix(raw.String(), "\r\n") {
					return raw.String(), nil
				}
			}
		}
	}

	return raw.String(), nil
}

// ReadDataBlock is used by the SMTP DATA command. It will read data from the connection until the terminator is sent.
func (r *Reader) ReadDataBlock() (string, error) {
	var dataBuffer bytes.Buffer

	for {
		dataResponse, err := r.Read()
		if err != nil {
			r.logger.Error("Error reading in DATA block", "error", err)

			return dataBuffer.String(), fmt.Errorf("Error reading in DATA block: %w", err)
		}

		dataBuffer.WriteString(dataResponse)
		terminatorPos := strings.Index(dataBuffer.String(), SMTP_DATA_TERMINATOR)

		if terminatorPos > -1 {
			break
		}
	}

	result := dataBuffer.String()
	result = result[:len(result)-3]

	return result, nil
}
