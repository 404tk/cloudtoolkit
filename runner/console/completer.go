package console

import (
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers"
	"github.com/404tk/cloudtoolkit/runner/catalog"
	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/go-prompt"
)

var modules = func() (m []prompt.Suggest) {
	for _, provider := range providers.Supported() {
		m = append(m, prompt.Suggest{Text: provider.Name, Description: provider.Desc})
	}
	return m
}()

var opt = []prompt.Suggest{
	{Text: "options", Description: "Display options"},
	{Text: "payloads", Description: "Display payloads"},
}

func Complete(d prompt.Document) []prompt.Suggest {
	return completeForMode(d, HelpModeRoot)
}

func actionCompleter(d prompt.Document) []prompt.Suggest {
	return completeForMode(d, HelpModeProvider)
}

func completeForMode(d prompt.Document, mode HelpMode) []prompt.Suggest {
	ctx := completionContextForMode(mode)
	args := completionArgs(d)
	word := d.GetWordBeforeCursor()

	switch mode {
	case HelpModeProvider:
		return providerSuggestions(ctx, args, word)
	case HelpModeShell:
		return shellSuggestions(ctx, args, word)
	default:
		return rootSuggestions(ctx, args, word)
	}
}

func rootSuggestions(ctx CompletionContext, args []string, word string) []prompt.Suggest {
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(commandSuggestionsForContext(ctx), word, true)
	}
	switch args[0] {
	case "help":
		return helpSuggestions(ctx, args, word)
	case "use":
		if len(args) == 2 {
			return prompt.FilterContains(modules, word, true)
		}
	case "sessions":
		return sessionsSuggestions(args, word)
	case "note":
		return noteSuggestions(args, word)
	}
	return []prompt.Suggest{}
}

func providerSuggestions(ctx CompletionContext, args []string, word string) []prompt.Suggest {
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(commandSuggestionsForContext(ctx), word, true)
	}
	switch args[0] {
	case "help":
		return helpSuggestions(ctx, args, word)
	case "use":
		if len(args) == 2 {
			return prompt.FilterContains(modules, word, true)
		}
	case "show":
		return showSuggestions(args, word)
	case "set":
		return setSuggestions(ctx, args, word)
	case "shell":
		return shellTargetSuggestions(ctx, args, word)
	case "sessions":
		return sessionsSuggestions(args, word)
	case "note":
		return noteSuggestions(args, word)
	}
	return []prompt.Suggest{}
}

func shellSuggestions(ctx CompletionContext, args []string, word string) []prompt.Suggest {
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(commandSuggestionsForContext(ctx), word, true)
	}
	switch args[0] {
	case "help":
		return helpSuggestions(ctx, args, word)
	}
	return []prompt.Suggest{}
}

func showSuggestions(args []string, word string) []prompt.Suggest {
	if len(args) == 2 {
		return prompt.FilterContains(showTopicSuggestionsData, word, true)
	}
	return []prompt.Suggest{}
}

func setSuggestions(ctx CompletionContext, args []string, word string) []prompt.Suggest {
	if len(args) == 2 {
		return prompt.FilterContains(optionSuggestions(ctx), word, true)
	}
	if len(args) != 3 {
		return []prompt.Suggest{}
	}
	switch args[1] {
	case utils.Payload:
		return prompt.FilterContains(payloadSuggestions(), word, true)
	case utils.Version:
		if _, ok := ctx.Config[utils.Version]; ok {
			return prompt.FilterContains(versionSuggestionsData, word, true)
		}
	case utils.Region:
		return prompt.FilterContains(getProviderRegionSuggestions(ctx.Provider), word, true)
	case utils.Metadata:
		return prompt.FilterContains(getPayloadMetadataSuggestions(ctx.Payload), word, true)
	}
	return []prompt.Suggest{}
}

func shellTargetSuggestions(ctx CompletionContext, args []string, word string) []prompt.Suggest {
	if len(args) == 2 {
		return prompt.FilterContains(getShellTargetSuggestions(ctx), word, true)
	}
	return []prompt.Suggest{}
}

func completionArgs(d prompt.Document) []string {
	text := d.TextBeforeCursor()
	args := strings.Fields(text)
	if strings.HasSuffix(text, " ") || strings.HasSuffix(text, "\t") {
		args = append(args, "")
	}
	return args
}

func helpSuggestions(ctx CompletionContext, args []string, word string) []prompt.Suggest {
	if len(args) == 2 {
		return prompt.FilterContains(helpTopicSuggestions(ctx), word, true)
	}
	if len(args) == 3 && (args[1] == "payload" || args[1] == "metadata") {
		return prompt.FilterContains(payloadSuggestions(), word, true)
	}
	return []prompt.Suggest{}
}

func helpTopicSuggestions(ctx CompletionContext) []prompt.Suggest {
	keys := helpTopicKeysForContext(ctx)
	suggestions := make([]prompt.Suggest, 0, len(keys))
	for _, key := range keys {
		if topic, ok := helpTopics[key]; ok {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        key,
				Description: topic.Summary,
			})
		}
	}
	return suggestions
}

func helpTopicKeysForContext(ctx CompletionContext) []string {
	keys := make([]string, 0, len(helpTopicOrder))
	seen := make(map[string]struct{}, len(helpTopicOrder))

	appendKey := func(key string) {
		if _, ok := helpTopics[key]; !ok {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	for _, key := range commandNamesForContext(ctx.Mode, ctx.DemoReplay, ctx.Provider) {
		appendKey(key)
	}
	if ctx.Mode == HelpModeProvider && ctx.DemoReplay {
		appendKey("demo")
	}
	appendKey("payload")
	appendKey("metadata")

	for _, key := range helpTopicOrder {
		appendKey(key)
	}
	return keys
}

func optionSuggestions(ctx CompletionContext) []prompt.Suggest {
	keys := []string{utils.Payload, utils.Metadata}
	seen := map[string]struct{}{
		utils.Payload:  {},
		utils.Metadata: {},
	}

	for _, key := range providerConfigKeys() {
		if _, ok := seen[key]; ok {
			continue
		}
		keys = append(keys, key)
		seen[key] = struct{}{}
	}
	for _, key := range []string{utils.Region, utils.Version} {
		if _, ok := ctx.Config[key]; !ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		keys = append(keys, key)
		seen[key] = struct{}{}
	}

	suggestions := make([]prompt.Suggest, 0, len(keys))
	for _, k := range keys {
		if v := catalog.OptionDescription(k); v != "" {
			suggestions = append(suggestions, prompt.Suggest{Text: k, Description: v})
		}
	}
	return suggestions
}

func payloadSuggestions() []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0, len(payloads.Visible()))
	for _, entry := range payloads.Visible() {
		suggestions = append(suggestions, prompt.Suggest{Text: entry.Name, Description: entry.Payload.Desc()})
	}
	return suggestions
}
