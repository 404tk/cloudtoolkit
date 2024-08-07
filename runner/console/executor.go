package console

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"

	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func Executor(s string) {
	// allow ctrl-c to generate signal
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		select {
		case <-c:
			cancel()
		}
	}()

	if s == "" {
		return
	}
	cmd, args := utils.ParseCmd(s)
	switch cmd {
	case "use":
		use(args)
	case "show":
		show(args)
	case "set":
		set(args)
	case "run":
		go func() { defer cancel(); run(ctx) }()
	case "shell":
		shell(args)
	case "sessions":
		sessions(args)
	case "note":
		note(args)
	case "help":
		help()
	case "clear":
		os.Stdout.Write([]byte("\033[2J\033[H"))
	case "exit", "quit":
		cache.SaveFile()
		os.Exit(0)
	default:
		logger.Error("Unsupported command:", cmd)
	}

	if cmd != "run" {
		return
	}
	select {
	case <-ctx.Done():
		return
	}
}

func show(args []string) {
	if len(args) != 1 {
		return
	}
	switch args[0] {
	case "options":
		fmt.Printf("\n%-10s\t%-60s\n", "Name", "Current Setting")
		fmt.Printf("%-10s\t%-60s\n", "----", "---------------")
		var tmplist []string
		for k := range config {
			tmplist = append(tmplist, k)
		}
		sort.Strings(tmplist)
		for _, k := range tmplist {
			if v, ok := config[k]; ok {
				fmt.Printf("%-15s\t%-60s\n", k, v)
			}
		}
	case "payloads":
		fmt.Printf("\n%-10s\t%-60s\n", "Payload", "Details")
		fmt.Printf("%-10s\t%-60s\n", "-------", "-------")
		for k, v := range payloads.Payloads {
			fmt.Printf("%-15s\t%-60s\n", k, v.Desc())
		}
	}
}

func set(args []string) {
	if len(args) < 2 {
		return
	}
	if _, ok := config[args[0]]; ok {
		if args[0] != utils.Provider {
			config[args[0]] = args[1]
			fmt.Printf("%s => %s\n", args[0], args[1])
		}
	}
	if args[0] == utils.Payload {
		switch args[1] {
		case "backdoor-user":
			config[utils.Metadata] = utils.BackdoorUser
		case "bucket-dump":
			config[utils.Metadata] = utils.BucketDump
		case "event-dump":
			config[utils.Metadata] = utils.EventDump
		default:
			config[utils.Metadata] = ""
		}
	}
}

func run(ctx context.Context) {
	if v, ok := payloads.Payloads[config[utils.Payload]]; ok {
		v.Run(ctx, config)
	} else {
		logger.Error("Please type `show payloads` to confirm the required payload.")
	}
}
