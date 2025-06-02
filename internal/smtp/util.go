// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"strings"
)

/*
GetCommandValue splits an input by colon (:) and returns the right hand side.
If there isn't a split, or a missing colon, an InvalidCommandFormatError is
returned.
*/
func GetCommandValue(streamInput, command, delimiter string) (string, error) {
	split := strings.Split(streamInput, delimiter)

	if len(split) < 2 {
		return "", InvalidCommandFormat(command)
	}

	return strings.TrimSpace(strings.Join(split[1:], "")), nil
}

/*
IsValidCommand returns an error if the input stream does not contain the expected command.
The input and expected commands are lower cased, as we do not care about
case when comparing.
*/
func IsValidCommand(streamInput, expectedCommand string) error {
	check := strings.Index(strings.ToLower(streamInput), strings.ToLower(expectedCommand))

	if check < 0 {
		return InvalidCommand(expectedCommand)
	}

	return nil
}
