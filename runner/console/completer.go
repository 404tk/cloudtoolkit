package console

import (
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/plugins"
	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/c-bata/go-prompt"
)

var core = []prompt.Suggest{
	{Text: "help", Description: "help menu"},
	{Text: "use", Description: "use module"},
	{Text: "sessions", Description: "list cache credential"},
	{Text: "clear", Description: "clear screen"},
	{Text: "exit", Description: "exit console"},
}

var action = []prompt.Suggest{
	{Text: "show", Description: "show options"},
	{Text: "set", Description: "set option"},
	{Text: "run", Description: "run job"},
}

var modules = func() (m []prompt.Suggest) {
	for k, v := range plugins.Providers {
		m = append(m, prompt.Suggest{Text: k, Description: v.Desc()})
	}
	return m
}()

var optionsDesc = map[string]string{
	// utils.Provider:              "Vendor Name",
	utils.Payload:               "Module Name (Default: cloudlist)",
	utils.AccessKey:             "Key ID",
	utils.SecretKey:             "Secret",
	utils.SecurityToken:         "Securit Token (Optional)",
	utils.Region:                "Region (Default: all)",
	utils.Version:               "International or custom edition (Optional)",
	utils.AzureClientId:         "Key ID",
	utils.AzureClientSecret:     "Secret",
	utils.AzureTenantId:         "Tenant ID",
	utils.AzureSubscriptionId:   "Subscription ID (Optional)",
	utils.GCPserviceAccountJSON: "GCP Credential encoded through Base64",
	utils.Metadata:              "Set the payload with additional arguments (Optional)",
	utils.Save:                  "Save log file",
}

var opt = []prompt.Suggest{
	{Text: "options", Description: "Display options"},
	{Text: "payloads", Description: "Display payloads"},
}

func Complete(d prompt.Document) []prompt.Suggest {
	args := strings.Split(d.TextBeforeCursor(), " ")
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(core, d.GetWordBeforeCursor(), true)
	}
	switch args[0] {
	case "use":
		if len(args) == 2 {
			return prompt.FilterContains(modules, d.GetWordBeforeCursor(), true)
		}
	}
	return []prompt.Suggest{}
}

func actionCompleter(d prompt.Document) []prompt.Suggest {
	args := strings.Split(d.TextBeforeCursor(), " ")
	if len(args) <= 1 {
		return prompt.FilterHasPrefix(append(core, action...), d.GetWordBeforeCursor(), true)
	}
	switch args[0] {
	case "use":
		if len(args) == 2 {
			return prompt.FilterContains(modules, d.GetWordBeforeCursor(), true)
		}
	case "show":
		if len(args) == 2 {
			return prompt.FilterContains(opt, d.GetWordBeforeCursor(), true)
		}
	case "set":
		if len(args) == 2 {
			getOpt := func() (p []prompt.Suggest) {
				for k := range config {
					if v, ok := optionsDesc[k]; ok { // && k != utils.Provider
						p = append(p, prompt.Suggest{Text: k, Description: v})
					}
				}
				return
			}()
			return prompt.FilterContains(getOpt, d.GetWordBeforeCursor(), true)
		}
		if len(args) == 3 && args[1] == utils.Payload {
			getPayloads := func() (p []prompt.Suggest) {
				for k, v := range payloads.Payloads {
					p = append(p, prompt.Suggest{Text: k, Description: v.Desc()})
				}
				return
			}()
			return prompt.FilterContains(getPayloads, d.GetWordBeforeCursor(), true)
		}
		if len(args) == 3 && args[1] == utils.Version {
			var versions = []prompt.Suggest{
				{Text: "Global", Description: "International Edition"},
				{Text: "China", Description: "Chinese Edition"},
			}
			return prompt.FilterContains(versions, d.GetWordBeforeCursor(), true)
		}
	}
	return []prompt.Suggest{}
}
