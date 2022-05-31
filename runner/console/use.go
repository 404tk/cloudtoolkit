package console

import (
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/c-bata/go-prompt"
)

var config = schema.OptionBlock{
	utils.Provider:     "",
	utils.AccessKey:    "",
	utils.SecretKey:    "",
	utils.SessionToken: "",
	utils.Region:       "",
	utils.Version:      "",
}

func Use(args []string) {
	if len(args) < 1 {
		return
	}
	for _, m := range modules {
		if m.Text == args[0] {
			loadModule(args[0])
			return
		}
	}
	fmt.Println("[Error] Unsupported module:", args[0])
}

func loadModule(m string) {
	config[utils.Provider] = m
	p := prompt.New(
		Executor,
		actionCompleter,
		prompt.OptionPrefix(fmt.Sprintf("ctk > %s > ", m)),
		prompt.OptionInputTextColor(prompt.White),
	)
	p.Run()
}
