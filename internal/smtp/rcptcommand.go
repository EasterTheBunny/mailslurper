// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"log/slog"
	"net/mail"

	"github.com/adampresley/webframework/sanitizer"

	"github.com/mailslurper/mailslurper/v2/internal/mailslurper"
	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// RcptCommandExecutor process the RCPT TO command.
type RcptCommandExecutor struct {
	emailValidationService mailslurper.EmailValidationProvider
	logger                 *slog.Logger
	reader                 *Reader
	writer                 *Writer
	xssService             sanitizer.IXSSServiceProvider
}

// NewRcptCommandExecutor creates a new struct.
func NewRcptCommandExecutor(
	logger *slog.Logger,
	reader *Reader,
	writer *Writer,
	emailValidationService mailslurper.EmailValidationProvider,
	xssService sanitizer.IXSSServiceProvider,
) *RcptCommandExecutor {
	return &RcptCommandExecutor{
		emailValidationService: emailValidationService,
		logger:                 logger,
		reader:                 reader,
		writer:                 writer,
		xssService:             xssService,
	}
}

// Process handles the RCPT TO command. This command tells us who the recipient is.
func (e *RcptCommandExecutor) Process(streamInput string, mailItem *model.MailItem) error {
	var (
		to           string
		toComponents *mail.Address
		err          error
	)

	if err = IsValidCommand(streamInput, "RCPT TO"); err != nil {
		return err
	}

	if to, err = GetCommandValue(streamInput, "RCPT TO", ":"); err != nil {
		return err
	}

	if toComponents, err = e.emailValidationService.GetEmailComponents(to); err != nil {
		return mailslurper.InvalidEmail(to)
	}

	to = e.xssService.SanitizeString(toComponents.Address)

	if !e.emailValidationService.IsValidEmail(to) {
		return mailslurper.InvalidEmail(to)
	}

	mailItem.ToAddresses = append(mailItem.ToAddresses, to)
	e.writer.SendOkResponse()

	return nil
}
