package console

import (
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/plugins"
	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/go-prompt"
)

var core = []prompt.Suggest{
	{Text: "help", Description: "help menu"},
	{Text: "use", Description: "use provider"},
	{Text: "sessions", Description: "list cache credential"},
	{Text: "note", Description: "add remarks to the session"},
	{Text: "clear", Description: "clear screen"},
	{Text: "exit", Description: "exit console"},
}

var action = []prompt.Suggest{
	{Text: "show", Description: "show options or payloads"},
	{Text: "set", Description: "set option or parameter"},
	{Text: "run", Description: "run selected payload"},
	{Text: "shell", Description: "open instance-cmd-check session"},
}

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
	args := completionArgs(d)
	word := d.GetWordBeforeCursor()
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(core, word, true)
	}
	switch args[0] {
	case "help":
		return helpSuggestions(args, word)
	case "use":
		if len(args) == 2 {
			return prompt.FilterContains(modules, word, true)
		}
	}
	return []prompt.Suggest{}
}

func actionCompleter(d prompt.Document) []prompt.Suggest {
	args := completionArgs(d)
	word := d.GetWordBeforeCursor()
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(append(core, action...), word, true)
	}
	switch args[0] {
	case "help":
		return helpSuggestions(args, word)
	case "use":
		if len(args) == 2 {
			return prompt.FilterContains(modules, word, true)
		}
	case "show":
		if len(args) == 2 {
			return prompt.FilterContains(opt, word, true)
		}
	case "set":
		if len(args) == 2 {
			return prompt.FilterContains(optionSuggestions(), word, true)
		}
		if len(args) == 3 && args[1] == utils.Payload {
			return prompt.FilterContains(payloadSuggestions(), word, true)
		}
		if len(args) == 3 && args[1] == utils.Version {
			var versions = []prompt.Suggest{
				{Text: "Intl", Description: "International Edition"},
				{Text: "China", Description: "Chinese Edition"},
			}
			return prompt.FilterContains(versions, word, true)
		}
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

func optionSuggestions() []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0, len(config))
	for k := range config {
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
