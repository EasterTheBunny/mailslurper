// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package app

import (
	"fmt"
	"log/slog"

	"github.com/mailslurper/mailslurper/v2/internal/model"
)

/*
A DatabaseReceiver takes a MailItem and writes it to a database
*/
type DatabaseReceiver struct {
	database IStorage
	logger   *slog.Logger
}

/*
NewDatabaseReceiver creates a new DatabaseReceiver object
*/
func NewDatabaseReceiver(database IStorage, logger *slog.Logger) DatabaseReceiver {
	return DatabaseReceiver{
		database: database,
		logger:   logger,
	}
}

/*
Receive takes a MailItem and writes it to the provided storage engine
*/
func (receiver DatabaseReceiver) Receive(mailItem *model.MailItem) error {
	var err error
	var newID string

	if newID, err = receiver.database.StoreMail(mailItem); err != nil {
		receiver.logger.Error(fmt.Sprintf("There was an error while storing your mail item: %s", err.Error()))
		return err
	}

	receiver.logger.Info(fmt.Sprintf("Mail item %s written", newID))

	return nil
}
