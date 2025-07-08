// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package model

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/adampresley/webframework/sanitizer"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/gofrs/uuid"
)

// MailItem is a struct describing a parsed mail item. This is populated after an incoming client connection has
// finished sending mail data to this server.
type MailItem struct {
	ID               uuid.UUID             `db:"id" json:"id"`
	DateSent         string                `db:"dateSent" json:"dateSent"`
	FromAddress      string                `db:"fromAddress" json:"fromAddress"`
	ToAddresses      MailAddressCollection `db:"toAddresses" json:"toAddresses"`
	Subject          string                `db:"subject" json:"subject"`
	XMailer          string                `db:"xmailer" json:"xmailer"`
	MIMEVersion      string                `db:"mimeVersion" json:"mimeVersion"`
	Body             string                `db:"body" json:"body"`
	ContentType      string                `db:"contentType" json:"contentType"`
	Boundary         string                `db:"boundary" json:"boundary"`
	TransferEncoding string                `db:"transferEncoding" json:"transferEncoding"`

	Attachments []*Attachment `has_many:"attachment" json:"-"`
	CreatedAt   time.Time     `db:"created_at" json:"-"`
	UpdatedAt   time.Time     `db:"updated_at" json:"-"`

	Message           *SMTPMessagePart `db:"-" json:"-"`
	InlineAttachments []*Attachment    `db:"-" json:"-"`
	TextBody          string           `db:"-" json:"-"`
	HTMLBody          string           `db:"-" json:"-"`
}

// NewEmptyMailItem creates an empty mail object.
func NewEmptyMailItem(logger *slog.Logger) *MailItem {
	id, _ := uuid.NewV4()

	result := &MailItem{
		ID:          id,
		ToAddresses: NewMailAddressCollection(),
		Attachments: make([]*Attachment, 0, 5),
		Message:     NewSMTPMessagePart(logger),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return result
}

// NewMailItem creates a new MailItem object.
func NewMailItem(
	id uuid.UUID,
	dateSent string,
	fromAddress string,
	toAddresses MailAddressCollection,
	subject, xMailer, body, contentType, boundary string,
	attachments []*Attachment,
	logger *slog.Logger,
) *MailItem {
	return &MailItem{
		ID:          id,
		DateSent:    dateSent,
		FromAddress: fromAddress,
		ToAddresses: toAddresses,
		Subject:     subject,
		XMailer:     xMailer,
		Body:        body,
		ContentType: contentType,
		Boundary:    boundary,
		Attachments: attachments,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),

		Message: NewSMTPMessagePart(logger),
	}
}

// Render implements the render.Renderer interface for use with chi-router.
func (_ *MailItem) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate,
// pop.ValidateAndUpdate) method.
func (m *MailItem) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.Validate(
		&validators.UUIDIsPresent{Name: "ID", Field: m.ID},
		&validators.StringIsPresent{Name: "FromAddress", Field: m.FromAddress},
		&validators.TimeIsPresent{Name: "CreatedAt", Field: m.CreatedAt},
		&validators.TimeIsPresent{Name: "UpdatedAt", Field: m.UpdatedAt},
	), nil
}

func (m *MailItem) Sanitize(xss sanitizer.IXSSServiceProvider) {
	m.Subject = xss.SanitizeString(m.Subject)
	m.XMailer = xss.SanitizeString(m.XMailer)
	m.Body = xss.SanitizeString(m.Body)

	for _, att := range m.Attachments {
		att.Sanitize(xss)
	}
}
