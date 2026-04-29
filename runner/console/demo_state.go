package console

import (
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/replay"
	tencentreplay "github.com/404tk/cloudtoolkit/pkg/providers/tencent/replay"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/go-prompt"
)

func resetDemoReplay() {
	replay.Disable()
	tencentreplay.Reset()
}

func enableDemoReplay(provider string) {
	replay.Enable(provider)
}

func isDemoReplayActive() bool {
	return replay.IsActive()
}

func isDemoReplayActiveForCurrentProvider() bool {
	if !isDemoReplayActive() || config == nil {
		return false
	}
	return replay.IsActiveForProvider(config[utils.Provider])
}

func isDemoRunHandledByProviderReplay() bool {
	if !isDemoReplayActiveForCurrentProvider() || config == nil {
		return false
	}
	return replay.SupportsProvider(config[utils.Provider])
}

func demoCommand() {
	provider := strings.TrimSpace(config[utils.Provider])
	if config == nil || provider == "" {
		logger.Error("Demo replay only works inside provider mode. Run `use <provider>` first.")
		return
	}
	if _, ok := replay.CredentialsFor(provider); !ok {
		logger.Error("Demo replay is not available for", provider)
		return
	}
	if isDemoReplayActiveForCurrentProvider() {
		logger.Error("Mock replay is already enabled. Use `exit` to return to the live provider session.")
		return
	}

	enableDemoReplay(provider)
	printDemoBanner(provider)

	p := prompt.New(
		demoExecutor,
		demoCompleter,
		prompt.OptionPrefix(mockPromptPrefix(provider)),
		prompt.OptionInputTextColor(prompt.White),
		sharedConsoleHistoryOption(),
	)
	consoleStack = append(consoleStack, currentConsole)
	currentConsole = p
	p.Run()
}

func demoExecutor(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}

	name, args := utils.ParseCmd(cmd)
	switch name {
	case "exit", "quit", "back":
		if len(args) != 0 {
			logger.Error("Use `exit`, `quit`, or `back` without arguments to leave mock mode.")
			return
		}
		closeDemoPrompt()
		return
	case "demo":
		logger.Error("Mock replay is already enabled. Use `exit` to return to the live provider session.")
		return
	case "use":
		logger.Error("Leave mock mode with `exit` before switching providers.")
		return
	case "note":
		logger.Error("Mock replay sessions do not support notes.")
		return
	}

	Executor(cmd)
}

func demoCompleter(d prompt.Document) []prompt.Suggest {
	return providerSuggestions(
		completionContextForMode(HelpModeProvider),
		completionArgs(d),
		d.GetWordBeforeCursor(),
	)
}

func closeDemoPrompt() {
	if len(consoleStack) == 0 {
		logger.Error("No live provider session to return to.")
		resetDemoReplay()
		return
	}

	provider := replay.ActiveProvider()
	prevConsole := consoleStack[len(consoleStack)-1]
	consoleStack = consoleStack[:len(consoleStack)-1]
	currentConsole = prevConsole
	resetDemoReplay()

	fmt.Println()
	fmt.Println("Mock replay disabled.")
	fmt.Printf("Returned to the live provider session for %s.\n", provider)
	fmt.Println()

	prevConsole.Run()
}

func printDemoBanner(provider string) {
	credentials, _ := replay.CredentialsFor(provider)
	printDemoSection("MOCK REPLAY MODE",
		"!!! USE THESE DEMO CREDENTIALS !!!",
		"----------------------------------",
		fmt.Sprintf("AccessKey: %s", credentials.AccessKey),
		fmt.Sprintf("SecretKey: %s", credentials.SecretKey),
	)
}

func printDemoSection(title string, lines ...string) {
	fmt.Println()
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", len(title)))
	fmt.Println()
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println()
}
