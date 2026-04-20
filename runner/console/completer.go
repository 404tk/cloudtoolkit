package console

import (
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/plugins"
	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/go-prompt"
)

var modules = func() (m []prompt.Suggest) {
	for k, v := range plugins.Providers {
		m = append(m, prompt.Suggest{Text: k, Description: v.Desc()})
	}
	return m
}()

var optionsDesc = map[string]string{
	// utils.Provider:              "Vendor Name",
	utils.Payload:               "Validation payload (Default: cloudlist)",
	utils.AccessKey:             "Key ID",
	utils.SecretKey:             "Secret",
	utils.SecurityToken:         "Security Token (Optional)",
	utils.Region:                "Region (Default: all)",
	utils.Version:               "International or custom edition (Optional)",
	utils.AzureClientId:         "Key ID",
	utils.AzureClientSecret:     "Secret",
	utils.AzureTenantId:         "Tenant ID",
	utils.AzureSubscriptionId:   "Subscription ID (Optional)",
	utils.GCPserviceAccountJSON: "GCP Credential encoded through Base64",
	utils.Metadata:              "Set the payload with additional arguments (Optional)",
}

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

func rootSuggestions(_ CompletionContext, args []string, word string) []prompt.Suggest {
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(rootCommandSuggestions, word, true)
	}
	switch args[0] {
	case "help":
		return helpSuggestions(args, word)
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
		return prompt.FilterHasPrefix(providerCommandSuggestions(), word, true)
	}
	switch args[0] {
	case "help":
		return helpSuggestions(args, word)
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

func shellSuggestions(_ CompletionContext, args []string, word string) []prompt.Suggest {
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(shellCommandSuggestions, word, true)
	}
	switch args[0] {
	case "help":
		return helpSuggestions(args, word)
	}
	return []prompt.Suggest{}
}

func showSuggestions(args []string, word string) []prompt.Suggest {
	if len(args) == 2 {
		return prompt.FilterContains(showTopicSuggestions(), word, true)
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

func helpSuggestions(args []string, word string) []prompt.Suggest {
	if len(args) == 2 {
		return prompt.FilterContains(helpTopicSuggestions(), word, true)
	}
	if len(args) == 3 && (args[1] == "payload" || args[1] == "metadata") {
		return prompt.FilterContains(payloadSuggestions(), word, true)
	}
	return []prompt.Suggest{}
}

func helpTopicSuggestions() []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0, len(helpTopicOrder))
	for _, key := range helpTopicOrder {
		if topic, ok := helpTopics[key]; ok {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        key,
				Description: topic.Summary,
			})
		}
	}
	return suggestions
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
		if v, ok := optionsDesc[k]; ok {
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
