// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package io

import "strings"

/*
StorageType defines types of database engines MailSlurper supports
*/
type StorageType int

const (
	STORAGE_MSSQL StorageType = iota
	STORAGE_SQLITE
	STORAGE_MYSQL
	STORAGE_POSTGRES
)

func GetDatabaseEngineFromName(engineName string) (StorageType, error) {
	switch strings.ToLower(engineName) {
	case "mssql":
		return STORAGE_MSSQL, nil
	case "mysql":
		return STORAGE_MYSQL, nil
	case "sqlite":
		return STORAGE_SQLITE, nil
	case "postgres":
		return STORAGE_POSTGRES, nil
	}

	return 0, ErrInvalidDatabaseDialect
}

func IsValidStorageType(storageType string) bool {
	_, err := GetDatabaseEngineFromName(storageType)
	if err != nil {
		return false
	}

	return true
}

func NeedDBHost(storageType string) bool {
	if strings.ToLower(storageType) == "sqlite" {
		return false
	}

	return true
}
