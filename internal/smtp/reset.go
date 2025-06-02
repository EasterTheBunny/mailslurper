// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// ResetCommandExecutor process the command RSET.
type ResetCommandExecutor struct {
	logger *slog.Logger
	writer *Writer
}

// NewResetCommandExecutor creates a new struct.
func NewResetCommandExecutor(logger *slog.Logger, writer *Writer) *ResetCommandExecutor {
	return &ResetCommandExecutor{
		logger: logger,
		writer: writer,
	}
}

// Process handles the RSET command.
func (e *ResetCommandExecutor) Process(streamInput string, mailItem *model.MailItem) error {
	if strings.ToLower(streamInput) != "rset" {
		return fmt.Errorf("Invalid RSET command")
	}

	// Overwrite current mail object with an empty one
	*mailItem = *model.NewEmptyMailItem(e.logger)

	return e.writer.SendOkResponse()
}
