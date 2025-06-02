package main

import "github.com/mailslurper/mailslurper/v2/cmd"

var (
	// Version of the MailSlurper Server application
	SERVER_VERSION string = "2.0.0-rc1"
)

func main() {
	cmd.Execute(SERVER_VERSION)
}
