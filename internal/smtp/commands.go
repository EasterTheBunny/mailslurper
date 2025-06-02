// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"fmt"
	"strings"
)

// Command represents a command issued over a TCP connection.
type Command int

const (
	NONE Command = iota
	RCPT Command = iota
	MAIL Command = iota
	HELO Command = iota
	RSET Command = iota
	DATA Command = iota
	QUIT Command = iota
	NOOP Command = iota
)

// Commands is a map of SMTP command strings to their int representation. This is primarily used because there can be
// more than one command to do the same things. For example, a client can send "helo" or "ehlo" to initiate the
// handshake.
var Commands = map[string]Command{
	"helo":      HELO,
	"ehlo":      HELO,
	"rcpt to":   RCPT,
	"mail from": MAIL,
	"send":      MAIL,
	"rset":      RSET,
	"quit":      QUIT,
	"data":      DATA,
	"noop":      NOOP,
}

// CommandsToStrings is a friendly string representations of commands. Useful in error reporting.
var CommandsToStrings = map[Command]string{
	HELO: "HELO",
	RCPT: "RCPT TO",
	MAIL: "SEND",
	RSET: "RSET",
	QUIT: "QUIT",
	DATA: "DATA",
	NOOP: "NOOP",
}

// GetCommandFromString takes a string and returns the integer command representation. For example if the string
// contains "DATA" then the value 1 (the constant DATA) will be returned.
func GetCommandFromString(input string) (Command, error) {
	result := NONE
	input = strings.ToLower(input)

	if input == "" {
		return result, nil
	}

	for key, value := range Commands {
		if strings.Index(input, key) == 0 {
			result = value
			break
		}
	}

	if result == NONE {
		return result, fmt.Errorf("Command '%s' not found", input)
	}

	return result, nil
}

// String returns the string representation of a command.
func (smtpCommand Command) String() string {
	return CommandsToStrings[smtpCommand]
}
