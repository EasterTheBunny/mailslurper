// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package mailslurper

import "github.com/mailslurper/mailslurper/v2/internal/model"

/*
An IMailItemReceiver defines an interface where the implementing object
can take a MailItem and do something with it, like write to a database,
etc...
*/
type IMailItemReceiver interface {
	Receive(mailItem *model.MailItem) error
}
