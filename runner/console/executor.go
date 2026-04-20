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
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/confirm"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func Executor(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	rememberConsoleCommand(s)
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
			help(args)
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
// cloud state. Read-only payloads (cloud asset inventory via `cloudlist`,
// bucket-check, and event-check in dump mode) bypass the prompt.
// instance-cmd-check is skipped here so the shell REPL, which enters a single
// confirmation at session start, does not prompt on every keystroke.
func confirmIfSensitive(config map[string]string) bool {
	payload := payloads.ResolveName(config[utils.Payload])
	metadata := config[utils.Metadata]
	parts := argparse.Split(metadata)
	provider := config[utils.Provider]

	switch payload {
	case "iam-user-check":
		if len(parts) < 2 {
			return true
		}
		return confirm.Ask("iam-user-check."+parts[0], provider, parts[1])
	case "rds-account-check":
		if len(parts) < 2 {
			return true
		}
		return confirm.Ask("rds-account-check."+parts[0], provider, parts[1])
	case "event-check":
		if len(parts) < 1 || parts[0] != "whitelist" {
			return true
		}
		resource := ""
		if len(parts) >= 2 {
			resource = parts[1]
		}
		return confirm.Ask("event-check.whitelist", provider, resource)
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
		fmt.Printf("\n%-24s\t%-60s\n", "Payload", "Details")
		fmt.Printf("%-24s\t%-60s\n", "-------", "-------")
		for _, entry := range payloads.Visible() {
			fmt.Printf("%-24s\t%-60s\n", entry.Name, entry.Payload.Desc())
		}
	}
}

func set(args []string) {
	if len(args) < 2 {
		return
	}
	if config == nil {
		return
	}
	key := args[0]
	if key == utils.Provider {
		return
	}

	if _, ok := config[key]; ok || key == utils.Metadata || key == utils.Payload {
		value := args[1]
		if key == utils.Payload {
			value = payloads.ResolveName(value)
		}
		config[key] = value
		fmt.Printf("%s => %s\n", key, value)

		if key == utils.Metadata && payloads.ResolveName(config[utils.Payload]) == "instance-cmd-check" {
			if target := shellTargetFromMetadata(value); target != "" {
				rememberShellTarget(target, config[utils.Provider], "instance-cmd-check metadata")
			}
		}
	}
	if key == utils.Payload {
		switch config[utils.Payload] {
		case "iam-user-check":
			config[utils.Metadata] = utils.IAMUserCheck
		case "bucket-check":
			config[utils.Metadata] = utils.BucketCheck
		case "event-check":
			config[utils.Metadata] = utils.EventCheck
		default:
			config[utils.Metadata] = ""
		}
		if target := shellTargetFromConfig(config); target != "" {
			rememberShellTarget(target, config[utils.Provider], "payload metadata")
		}
	}
}

func run(ctx context.Context) {
	if v, name, ok := payloads.Lookup(config[utils.Payload]); ok {
		config[utils.Payload] = name
		v.Run(ctx, config)
	} else {
		logger.Error("Please type `show payloads` to confirm the required payload.")
	}
}
