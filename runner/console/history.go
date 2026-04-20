package console

import (
	"strings"

	"github.com/404tk/go-prompt"
)

var sharedConsoleHistory []string

func rememberConsoleCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}
	sharedConsoleHistory = append(sharedConsoleHistory, cmd)
}

func sharedConsoleHistorySnapshot() []string {
	return append([]string(nil), sharedConsoleHistory...)
}

func sharedConsoleHistoryOption() prompt.Option {
	return prompt.OptionHistory(sharedConsoleHistorySnapshot())
}
