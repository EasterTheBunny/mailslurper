// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

// Responses that are sent to SMTP clients.
const (
	SMTP_CRLF                     string = "\r\n"
	SMTP_DATA_TERMINATOR          string = "\r\n.\r\n"
	SMTP_WELCOME_MESSAGE          string = "220 Welcome to MailSlurper!"
	SMTP_CLOSING_MESSAGE          string = "221 Bye"
	SMTP_OK_MESSAGE               string = "250 Ok"
	SMTP_DATA_RESPONSE_MESSAGE    string = "354 End data with <CR><LF>.<CR><LF>"
	SMTP_HELLO_RESPONSE_MESSAGE   string = "250 Hello. How very nice to meet you!"
	SMTP_ERROR_TRANSACTION_FAILED string = "554 Transaction failed"
)

// SMTPWorkerState defines states that a worker may be in. Typically a worker starts IDLE, the moves to WORKING, finally
// going to either DONE or ERROR.
type SMTPWorkerState int

const (
	SMTP_WORKER_IDLE    SMTPWorkerState = 0
	SMTP_WORKER_WORKING SMTPWorkerState = 1
	SMTP_WORKER_DONE    SMTPWorkerState = 100
	SMTP_WORKER_ERROR   SMTPWorkerState = 101

	RECEIVE_BUFFER_LEN         = 1024
	CONNECTION_TIMEOUT_MINUTES = 10
	COMMAND_TIMEOUT_SECONDS    = 5
)
