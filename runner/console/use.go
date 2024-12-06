package console

import (
	"errors"
	"fmt"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/go-prompt"
)

var config map[string]string

func use(args []string) {
	if len(args) < 1 {
		logger.Error("Example: use alibaba")
		return
	}
	if err := loadModule(args[0]); err != nil {
		logger.Error(err)
	}
}

func loadModule(m string) error {
	switch m {
	case "alibaba", "tencent", "huawei", "aws", "volcengine":
		config = loadConfig1()
	case "azure":
		config = loadConfig2()
	case "gcp":
		config = loadConfig3()
	default:
		return errors.New("Unsupported module: " + m)
	}

	config[utils.Provider] = m
	config[utils.Payload] = "cloudlist" // Default use cloudlist

	p := prompt.New(
		Executor,
		actionCompleter,
		prompt.OptionPrefix(fmt.Sprintf("ctk > %s > ", m)),
		prompt.OptionInputTextColor(prompt.White),
	)
	currentConsole = p
	p.Run()
	return nil
}

func loadConfig1() map[string]string {
	return map[string]string{
		utils.AccessKey:     "",
		utils.SecretKey:     "",
		utils.SecurityToken: "",
		utils.Region:        "all",    // Default enumerate all
		utils.Version:       "Global", // Default select International Edition
	}
}

func loadConfig2() map[string]string {
	return map[string]string{
		utils.AzureClientId:       "",
		utils.AzureClientSecret:   "",
		utils.AzureTenantId:       "",
		utils.AzureSubscriptionId: "",
	}
}

func loadConfig3() map[string]string {
	return map[string]string{utils.GCPserviceAccountJSON: ""}
}
