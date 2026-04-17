package console

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/confirm"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func Executor(s string) {
	if s == "" {
		return
	}
	cmd, args := utils.ParseCmd(s)

	// Only payload `run` needs a cancellable context with timeout + SIGINT wiring.
	// Other commands return quickly and don't benefit from the plumbing.
	if cmd != "run" {
		switch cmd {
		case "use":
			use(args)
		case "show":
			show(args)
		case "set":
			set(args)
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
		return
	}

	if !confirmIfSensitive(config) {
		logger.Info("Cancelled.")
		return
	}

	timeout := utils.RunTimeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer signal.Stop(c)

	done := make(chan struct{})
	go func() {
		defer close(done)
		run(ctx)
	}()

	select {
	case <-done:
	case <-c:
		logger.Info("Interrupted, cancelling...")
		cancel()
		<-done
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			logger.Error(fmt.Sprintf("Run timed out after %s", timeout))
		}
		<-done
	}
}

// confirmIfSensitive prompts the user before dispatching payloads that mutate
// cloud state. Read-only payloads (cloudlist, bucket-dump, event-dump.dump)
// bypass the prompt. exec-command is intentionally skipped here so the shell
// REPL, which enters a single confirmation at session start, does not prompt
// on every keystroke.
func confirmIfSensitive(config map[string]string) bool {
	payload := config[utils.Payload]
	metadata := config[utils.Metadata]
	parts := strings.Fields(metadata)
	provider := config[utils.Provider]

	switch payload {
	case "backdoor-user":
		if len(parts) < 2 {
			return true
		}
		return confirm.Ask("backdoor-user."+parts[0], provider, parts[1])
	case "database-account":
		if len(parts) < 2 {
			return true
		}
		return confirm.Ask("database-account."+parts[0], provider, parts[1])
	case "event-dump":
		if len(parts) < 1 || parts[0] != "whitelist" {
			return true
		}
		target := ""
		if len(parts) >= 2 {
			target = parts[1]
		}
		return confirm.Ask("event-dump.whitelist", provider, target)
	}
	return true
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
