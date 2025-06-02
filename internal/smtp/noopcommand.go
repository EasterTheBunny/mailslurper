// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"log/slog"

	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// NoopCommandExecutor process the command NOOP.
type NoopCommandExecutor struct {
	logger *slog.Logger
	writer *Writer
}

// NewNoopCommandExecutor creates a new struct.
func NewNoopCommandExecutor(logger *slog.Logger, writer *Writer) *NoopCommandExecutor {
	return &NoopCommandExecutor{
		logger: logger,
		writer: writer,
	}
}

// Process handles the NOOP command.
func (e *NoopCommandExecutor) Process(streamInput string, mailItem *model.MailItem) error {
	var err error

	if err = IsValidCommand(streamInput, "NOOP"); err != nil {
		return err
	}

	// log the command, and do nothing
	e.logger.Debug("NOOP command received")

	// Response: 250 Ok
	return e.writer.SendOkResponse()
}
