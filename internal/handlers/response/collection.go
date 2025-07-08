// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package response

import (
	"net/http"

	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// MailCollectionResponse is sent in response to getting a collection of mail items.
type MailCollectionResponse struct {
	MailItems    []model.MailItem `json:"mailItems"`
	TotalPages   int              `json:"totalPages"`
	TotalRecords int              `json:"totalRecords"`
}

// Render implements the render.Renderer interface for use with chi-router.
func (_ *MailCollectionResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
