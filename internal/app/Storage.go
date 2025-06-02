// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package app

import (
	"log/slog"

	slurperio "github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/persistence"
)

/*
ConnectToStorage establishes a connection to the configured database engine and returns
an object.
*/
func ConnectToStorage(storageType slurperio.StorageType, connectionInfo *slurperio.ConnectionInformation, logger *slog.Logger) (IStorage, error) {
	var err error
	var storageHandle IStorage

	logger.Info("Connecting to database")

	switch storageType {
	case slurperio.STORAGE_SQLITE:
		storageHandle = persistence.NewSQLiteStorage(connectionInfo, logger)
	case slurperio.STORAGE_MSSQL:
		storageHandle = persistence.NewMSSQLStorage(connectionInfo, logger)
	case slurperio.STORAGE_MYSQL:
		storageHandle = persistence.NewMySQLStorage(connectionInfo, logger)
	case slurperio.STORAGE_POSTGRES:
		storageHandle = persistence.NewPgSQLStorage(connectionInfo, logger)
	}

	if err = storageHandle.Connect(); err != nil {
		return storageHandle, err
	}

	if err = storageHandle.Create(); err != nil {
		return storageHandle, err
	}

	return storageHandle, nil
}
