package console

import (
	"strings"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/c-bata/go-prompt"
)

var core = []prompt.Suggest{
	{Text: "use", Description: "use module"},
	{Text: "clear", Description: "clear screen"},
	{Text: "exit", Description: "exit console"},
}

var action = []prompt.Suggest{
	{Text: "show", Description: "show options"},
	{Text: "set", Description: "set option"},
	{Text: "run", Description: "run job"},
}

var modules = []prompt.Suggest{
	{Text: "aws", Description: "Amazon Web Service"},
	{Text: "gcp", Description: "Google Cloud Platform"},
	{Text: "azure", Description: "Microsoft Azure"},
	{Text: "alibaba", Description: "Alibaba Cloud"},
	{Text: "tencent", Description: "Tencent Cloud"},
	{Text: "huawei", Description: "Huawei Cloud"},
}

/*
var enumerate = []prompt.Suggest{
	// {Text: utils.Provider, Description: "Vendor name"},
	{Text: utils.AccessKey, Description: "key ID"},
	{Text: utils.SecretKey, Description: "Secret"},
	{Text: utils.SessionToken, Description: "session token(optional)"},
	{Text: utils.Region, Description: "Region(optional)"},
	{Text: utils.Version, Description: "International or custom edition(optional)"},
}
*/

func getOpt() []prompt.Suggest {
	var enumerate = []prompt.Suggest{}
	for k := range config {
		if k != utils.Provider {
			enumerate = append(enumerate, prompt.Suggest{Text: k})
		}
	}
	return enumerate
}

var opt = []prompt.Suggest{
	{Text: "options", Description: "Display options"},
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
			return prompt.FilterContains(getOpt(), d.GetWordBeforeCursor(), true)
		}
	}
	return []prompt.Suggest{}
}
