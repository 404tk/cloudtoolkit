package console

import (
	"github.com/404tk/cloudtoolkit/runner/catalog"
	"github.com/404tk/go-prompt"
)

func defaultProviderConfig(provider string) (map[string]string, bool) {
	return catalog.DefaultProviderConfig(provider)
}

func startProviderConsole(provider string) {
	p := prompt.New(
		Executor,
		actionCompleter,
		prompt.OptionPrefix(providerPromptPrefix(provider)),
		prompt.OptionInputTextColor(prompt.White),
		sharedConsoleHistoryOption(),
	)
	currentConsole = p
	p.Run()
}
