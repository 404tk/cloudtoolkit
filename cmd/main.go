package main

import (
	"os"
	"os/signal"
	"syscall"

	_ "github.com/404tk/cloudtoolkit/pkg/plugins"
	"github.com/404tk/cloudtoolkit/runner"
	"github.com/404tk/cloudtoolkit/runner/console"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/go-prompt"
)

func main() {
	runner.ShowBanner()
	runner.InitConfig()

	// Flush credential cache on fatal signals so AK/SK captured this session
	// aren't lost. SIGINT during an active `run` is handled locally in the
	// executor; this handler covers SIGTERM and any SIGINT we didn't intercept.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)
	go func() {
		<-sig
		cache.SaveFile()
		os.Exit(130)
	}()

	p := prompt.New(
		console.Executor,
		console.Complete,
		prompt.OptionTitle("CloudToolKit"),
		prompt.OptionPrefix("ctk > "),
		prompt.OptionInputTextColor(prompt.White),
	)
	p.Run()
}
