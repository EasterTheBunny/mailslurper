// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package response

import "net/http"

// MailCountResponse is used to report the number of mail items in storage.
type MailCountResponse struct {
	MailCount int `json:"mailCount"`
}

// Render implements the render.Renderer interface for use with chi-router.
func (_ *MailCountResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
