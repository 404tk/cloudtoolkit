package console

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/confirm"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/go-prompt"
)

var consoleStack []*prompt.Prompt
var currentConsole *prompt.Prompt
var instanceId string

func shell(args []string) {
	if len(args) < 1 {
		logger.Error("Usage: shell <instance-id>")
		return
	}
	if !confirm.Ask("instance-cmd-check session", config[utils.Provider], args[0]) {
		logger.Info("Cancelled.")
		return
	}
	instanceId = args[0]
	rememberShellTarget(instanceId, config[utils.Provider], "shell command")
	config[utils.Payload] = "instance-cmd-check"
	p := prompt.New(
		shellExecutor,
		shellCompleter,
		prompt.OptionPrefix(fmt.Sprintf("[validation@%s ~]$ ", instanceId)),
		prompt.OptionInputTextColor(prompt.White),
	)
	consoleStack = append(consoleStack, currentConsole)
	currentConsole = p
	p.Run()
}

func shellExecutor(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}
	name, args := utils.ParseCmd(cmd)
	switch name {
	case "help":
		help(args)
		return
	case "clear":
		if len(args) == 0 {
			os.Stdout.Write([]byte("\033[2J\033[H"))
			return
		}
	case "exit", "quit", "back":
		if len(args) == 0 {
			closeShell()
			return
		}
	}

	cmd = base64.StdEncoding.EncodeToString([]byte(cmd))
	config[utils.Metadata] = fmt.Sprintf("%s %s", instanceId, cmd)
	run(context.TODO())
}

func closeShell() {
	if len(consoleStack) == 0 {
		logger.Error("No previous console")
		return
	}
	target := instanceId
	prevConsole := consoleStack[len(consoleStack)-1]
	consoleStack = consoleStack[:len(consoleStack)-1]
	currentConsole = prevConsole
	config[utils.Payload] = "cloudlist"
	instanceId = ""
	logger.Info(fmt.Sprintf("Validation session to %s closed.", target))
	prevConsole.Run()
}

func shellCompleter(d prompt.Document) []prompt.Suggest {
	return completeForMode(d, HelpModeShell)
}
