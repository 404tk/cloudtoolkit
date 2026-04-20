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
	show          Displays provider options or validation payloads
	set           Sets a provider option or payload parameter
	run           Runs the selected validation workflow
	shell         Opens an authorized instance-cmd-check session


### Examples

Select a cloud provider:

	use aws

Displays the options required by the provider:

	show options

Display the available validation payloads:

	show payloads

Select a validation payload:

	set payload iam-user-check

Select a cache credential:

	sessions -i 1

Use CloudToolKit only in owned, lab, or explicitly authorized environments.`
	fmt.Println(document)
}
