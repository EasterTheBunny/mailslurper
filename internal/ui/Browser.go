// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package ui

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/skratchdot/open-golang/open"

	"github.com/mailslurper/mailslurper/v2/internal/io"
)

/*
StartBrowser opens the user's default browser to the configured URL
*/
func StartBrowser(config *io.Config, logger *slog.Logger) {
	timer := time.NewTimer(time.Second)

	go func() {
		<-timer.C
		logger.Info(fmt.Sprintf("Opening web browser to http://%s:%d", config.Public.Address, config.Public.Port))

		err := open.Start(fmt.Sprintf("http://%s:%d", config.Public.Address, config.Public.Port))
		if err != nil {
			logger.Info(fmt.Sprintf("ERROR - Could not open browser - %s", err.Error()))
		}
	}()
}
