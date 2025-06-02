// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package app

import (
	"github.com/mailslurper/mailslurper/v2/internal/model"
	"github.com/mailslurper/mailslurper/v2/internal/persistence"
)

/*
IStorage defines an interface for structures that need to connect to
storage engines. They store and retrieve data for MailSlurper
*/
type IStorage interface {
	Connect() error
	Disconnect()
	Create() error

	GetAttachment(mailID, attachmentID string) (*model.Attachment, error)
	GetMailByID(id string) (*model.MailItem, error)
	GetMailMessageRawByID(id string) (string, error)
	GetMailCollection(offset, length int, mailSearch *persistence.MailSearch) ([]*model.MailItem, error)
	GetMailCount(mailSearch *persistence.MailSearch) (int, error)

	DeleteMailsAfterDate(startDate string) (int64, error)
	StoreMail(mailItem *model.MailItem) (string, error)
}
