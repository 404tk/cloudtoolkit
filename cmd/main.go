package main

import (
	_ "github.com/404tk/cloudtoolkit/pkg/plugins"
	"github.com/404tk/cloudtoolkit/runner"
	"github.com/404tk/cloudtoolkit/runner/console"
	"github.com/c-bata/go-prompt"
)

func main() {
	runner.ShowBanner()
	runner.InitConfig()
	p := prompt.New(
		console.Executor,
		console.Complete,
		prompt.OptionTitle("CloudToolKit"),
		prompt.OptionPrefix("ctk > "),
		prompt.OptionInputTextColor(prompt.White),
	)
	p.Run()
}
