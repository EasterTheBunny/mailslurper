// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package mailslurper

import "github.com/mailslurper/mailslurper/v2/internal/handlers/requests"

/*
DeleteMailRequest is used when requesting to delete mail
items.
*/
type DeleteMailRequest struct {
	PruneCode requests.PruneCode `json:"pruneCode" form:"pruneCoe"`
}
