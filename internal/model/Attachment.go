// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package model

import (
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/adampresley/webframework/sanitizer"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/gofrs/uuid"
)

// An Attachment is any content embedded in the mail data that is not considered the body.
type Attachment struct {
	ID          uuid.UUID `db:"id" json:"id"`
	MailID      string    `db:"mailId" json:"mailId"`
	MailItem    *MailItem `belongs_to:"mailitem" json:"-"`
	FileName    string    `db:"fileName" json:"fileName"`
	ContentType string    `db:"contentType" json:"contentType"`
	Contents    string    `db:"contents" json:"contents"`
	CreatedAt   time.Time `db:"created_at" json:"-"`
	UpdatedAt   time.Time `db:"updated_at" json:"-"`

	Headers *AttachmentHeader `db:"-" json:"headers"`
}

// NewAttachment creates a new Attachment object.
func NewAttachment(headers *AttachmentHeader, contents string, xss sanitizer.IXSSServiceProvider) *Attachment {
	id, _ := uuid.NewV4()

	return &Attachment{
		ID:        id,
		Headers:   headers,
		Contents:  contents,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// IsContentBase64 returns true/false if the content of this attachment resembles a base64 encoded string.
func (a *Attachment) IsContentBase64() bool {
	spaceKiller := func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}

		return r
	}

	trimmedContents := strings.Map(spaceKiller, a.Contents)

	if math.Mod(float64(len(trimmedContents)), 4.0) == 0 {
		matchResult, err := regexp.Match("^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$", []byte(trimmedContents))
		if err == nil {
			if matchResult {
				return true
			}
		}
	}

	return false
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate,
// pop.ValidateAndUpdate) method.
func (a *Attachment) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.Validate(
		&validators.UUIDIsPresent{Name: "ID", Field: a.ID},
		&validators.TimeIsPresent{Name: "CreatedAt", Field: a.CreatedAt},
		&validators.TimeIsPresent{Name: "UpdatedAt", Field: a.UpdatedAt},
	), nil
}

func (a *Attachment) Sanitize(xss sanitizer.IXSSServiceProvider) {
	a.FileName = xss.SanitizeString(a.FileName)
}
