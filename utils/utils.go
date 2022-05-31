package utils

import "strings"

func ParseCmd(s string) (cmd string, args []string) {
	items := strings.Split(s, " ")
	cmd = items[0]
	if len(items) > 1 {
		args = items[1:]
	}
	return
}
