package console

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/go-prompt"
)

var consoleStack []*prompt.Prompt
var currentConsole *prompt.Prompt
var instanceId string

func shell(args []string) {
	if len(args) < 1 {
		logger.Error("Usage: shell instance-id")
		return
	}
	instanceId = args[0]
	config[utils.Payload] = "exec-command"
	p := prompt.New(
		shellExecutor,
		shellCompleter,
		prompt.OptionPrefix(fmt.Sprintf("[shell@%s ~]$ ", instanceId)),
		prompt.OptionInputTextColor(prompt.White),
	)
	consoleStack = append(consoleStack, currentConsole)
	currentConsole = p
	p.Run()
}

func shellExecutor(cmd string) {
	if cmd == "" {
		return
	}
	switch cmd {
	case "clear":
		os.Stdout.Write([]byte("\033[2J\033[H"))
	case "exit", "quit", "back":
		if len(consoleStack) > 0 {
			prevConsole := consoleStack[len(consoleStack)-1]
			consoleStack = consoleStack[:len(consoleStack)-1]
			currentConsole = prevConsole
			config[utils.Payload] = "cloudlist"
			logger.Info(fmt.Sprintf("Connection to %s closed.", instanceId))
			prevConsole.Run()
		} else {
			logger.Error("No previous console")
		}
	default:
		cmd = base64.StdEncoding.EncodeToString([]byte(cmd))
		config[utils.Metadata] = fmt.Sprintf("%s %s", instanceId, cmd)
		run(context.TODO())
	}
}

func shellCompleter(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "back", Description: "Go back to previous console"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}
