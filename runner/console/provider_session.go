package console

import (
	"strings"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/go-prompt"
)

var standardProviderConfigDefaults = map[string]string{
	utils.AccessKey:     "",
	utils.SecretKey:     "",
	utils.SecurityToken: "",
	utils.Region:        "all",
	utils.Version:       "",
}

var providerConfigDefaults = map[string]map[string]string{
	"alibaba":    standardProviderConfigDefaults,
	"tencent":    standardProviderConfigDefaults,
	"huawei":     standardProviderConfigDefaults,
	"aws":        standardProviderConfigDefaults,
	"volcengine": standardProviderConfigDefaults,
	"jdcloud":    standardProviderConfigDefaults,
	"azure": {
		utils.AzureClientId:       "",
		utils.AzureClientSecret:   "",
		utils.AzureTenantId:       "",
		utils.AzureSubscriptionId: "",
	},
	"gcp": {
		utils.GCPserviceAccountJSON: "",
	},
}

func defaultProviderConfig(provider string) (map[string]string, bool) {
	defaults, ok := providerConfigDefaults[strings.TrimSpace(provider)]
	if !ok {
		return nil, false
	}
	config := make(map[string]string, len(defaults))
	for key, value := range defaults {
		config[key] = value
	}
	return config, true
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
