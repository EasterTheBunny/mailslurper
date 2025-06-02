// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"errors"
	"fmt"
)

var (
	ErrServerClosed = errors.New("server closed")
)

/*
An ConnectionExistsError is used to alert a client that there is already a
connection by this address cached
*/
type ConnectionExistsError struct {
	Address string
}

/*
ConnectionExists returns a new error object
*/
func ConnectionExists(address string) *ConnectionExistsError {
	return &ConnectionExistsError{
		Address: address,
	}
}

func (err *ConnectionExistsError) Error() string {
	return fmt.Sprintf("Connection on '%s' already exists", err.Address)
}

/*
An ConnectionNotExistsError is used to alert a client that the specified
connection is not in the ConnectionManager pool
*/
type ConnectionNotExistsError struct {
	Address string
}

/*
ConnectionNotExists returns a new error object
*/
func ConnectionNotExists(address string) *ConnectionNotExistsError {
	return &ConnectionNotExistsError{
		Address: address,
	}
}

func (err *ConnectionNotExistsError) Error() string {
	return fmt.Sprintf("Connection '%s' is not in the connection manager pool", err.Address)
}

/*
NoWorkerAvailableError is an error used when no worker is available to
service a SMTP connection request.
*/
type NoWorkerAvailableError struct{}

/*
NoWorkerAvailable returns a new instance of the No Worker Available error
*/
func NoWorkerAvailable() NoWorkerAvailableError {
	return NoWorkerAvailableError{}
}

func (err NoWorkerAvailableError) Error() string {
	return "No worker available. Timeout has been exceeded"
}

/*
An InvalidCommandFormatError is used to alert a client that the command passed in
has an invalid format
*/
type InvalidCommandFormatError struct {
	InvalidCommand string
}

/*
InvalidCommandFormat returns a new error object
*/
func InvalidCommandFormat(command string) *InvalidCommandFormatError {
	return &InvalidCommandFormatError{
		InvalidCommand: command,
	}
}

func (err *InvalidCommandFormatError) Error() string {
	return fmt.Sprintf("%s command format is invalid", err.InvalidCommand)
}

/*
An InvalidCommandError is used to alert a client that the command passed in
is invalid.
*/
type InvalidCommandError struct {
	InvalidCommand string
}

/*
InvalidCommand returns a new error object
*/
func InvalidCommand(command string) *InvalidCommandError {
	return &InvalidCommandError{
		InvalidCommand: command,
	}
}

func (err *InvalidCommandError) Error() string {
	return fmt.Sprintf("Invalid command %s", err.InvalidCommand)
}
