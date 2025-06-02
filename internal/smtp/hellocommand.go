// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// HelloCommandExecutor process the commands EHLO, HELO.
type HelloCommandExecutor struct {
	logger *slog.Logger
	reader *Reader
	writer *Writer
}

// NewHelloCommandExecutor creates a new struct.
func NewHelloCommandExecutor(logger *slog.Logger, reader *Reader, writer *Writer) *HelloCommandExecutor {
	return &HelloCommandExecutor{
		logger: logger,
		reader: reader,
		writer: writer,
	}
}

// Process handles the HELO greeting command.
func (e *HelloCommandExecutor) Process(streamInput string, mailItem *model.MailItem) error {
	lowercaseStreamInput := strings.ToLower(streamInput)

	commandCheck := (strings.Index(lowercaseStreamInput, "helo") + 1) + (strings.Index(lowercaseStreamInput, "ehlo") + 1)
	if commandCheck <= 0 {
		return fmt.Errorf("Invalid HELO command")
	}

	split := strings.Split(streamInput, " ")
	if len(split) < 2 {
		return fmt.Errorf("HELO command format is invalid")
	}

	return e.writer.SendHELOResponse()
}
