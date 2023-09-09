package console

import "fmt"

func help() {
	document := `
Core Commands
=============

	Command       Description
	-------       -----------
	help          Help menu
	use           Interact with a provider by name
	sessions      List cache credential and display information about credentials
	clear         Clear screen
	exit          Exit the console


Module Commands
===============

	Command       Description
	-------       -----------
	show          Displays options of a given type, or all payloads
	set           Sets a context-specific variable to a value
	run           Run the jobs
	shell         Run commands on the cloud host


### Examples

Select a cloud provider:

	use aws

Displays the options required by the provider:

	show options

Select a payload:

	set payload backdoor-user

Select a cache credential:

	sessions -i 1`
	fmt.Println(document)
}
