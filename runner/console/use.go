package console

import (
	"fmt"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/c-bata/go-prompt"
)

var config map[string]string

func Use(args []string) {
	if len(args) < 1 {
		return
	}
	for _, m := range modules {
		if m.Text == args[0] {
			loadModule(args[0])
			if config["save"] == "true" {
				utils.DoSave = true
				utils.CheckLogDir()
			} else {
				utils.DoSave = false
			}
			return
		}
	}
	fmt.Println("[Error] Unsupported module:", args[0])
}

func loadModule(m string) {
	switch m {
	case "azure":
		config = loadConfig2()
	case "gcp":
		config = loadConfig3()
	default:
		config = loadConfig1()
	}

	config[utils.Provider] = m
	config[utils.Payload] = "cloudlist" // Default use cloudlist
	config[utils.Save] = "true"         // Default save log file

	p := prompt.New(
		Executor,
		actionCompleter,
		prompt.OptionPrefix(fmt.Sprintf("ctk > %s > ", m)),
		prompt.OptionInputTextColor(prompt.White),
	)
	p.Run()
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
