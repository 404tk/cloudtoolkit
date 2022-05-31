package main

import (
	"github.com/404tk/cloudtoolkit/runner"
	"github.com/404tk/cloudtoolkit/runner/console"
	"github.com/c-bata/go-prompt"
)

func main() {
	runner.ShowBanner()
	p := prompt.New(
		console.Executor,
		console.Complete,
		prompt.OptionTitle("CloudToolKit"),
		prompt.OptionPrefix("ctk > "),
		prompt.OptionInputTextColor(prompt.White),
	)
	p.Run()
}
