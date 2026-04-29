package console

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/go-prompt"
)

func defaultProviderConfig(provider string) (map[string]string, bool) {
	return registry.DefaultConfig(provider)
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
