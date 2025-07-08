// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package app

import (
	"fmt"
	"log/slog"

	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// MailWriter stores a mail item to a persistance layer.
type MailWriter interface {
	StoreMail(mailItem *model.MailItem) error
}

// DatabaseReceiver takes a MailItem and writes it to a database.
type DatabaseReceiver struct {
	orm    MailWriter
	logger *slog.Logger
}

// NewDatabaseReceiver creates a new DatabaseReceiver object.
func NewDatabaseReceiver(orm MailWriter, logger *slog.Logger) DatabaseReceiver {
	return DatabaseReceiver{
		orm:    orm,
		logger: logger,
	}
}

// Receive takes a MailItem and writes it to the provided storage engine.
func (r DatabaseReceiver) Receive(mailItem *model.MailItem) error {
	if err := r.orm.StoreMail(mailItem); err != nil {
		r.logger.Error(fmt.Sprintf("There was an error while storing your mail item: %s", err.Error()))

		return err
	}

	// r.logger.Info(fmt.Sprintf("Mail item %s written", newID))

	return nil
}
