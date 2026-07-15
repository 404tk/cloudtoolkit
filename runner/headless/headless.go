package headless

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"

	"github.com/404tk/cloudtoolkit/pkg/providers"
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/runner"
	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/confirm"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func Run(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return RunContext(ctx, args)
}

// RunContext executes a headless command with a caller-controlled cancellation
// and deadline boundary.
func RunContext(ctx context.Context, args []string) int {
	if ctx == nil {
		ctx = context.Background()
	}
	if wantsHelp(args) {
		return writeHelp()
	}
	flags, remaining, err := parseFlags(args)
	if err != nil {
		return fail(flags.JSON, exitConfigError, err)
	}
	if flags.Describe && len(remaining) == 0 {
		return writeVersion(flags.JSON)
	}
	if len(remaining) == 0 {
		return fail(flags.JSON, exitConfigError, fmt.Errorf("missing command"))
	}

	logger.SetOutputs(os.Stderr, os.Stderr)
	defer logger.SetOutputs(os.Stdout, os.Stderr)
	processbar.SetOutput(os.Stderr)
	defer processbar.SetOutput(nil)
	debugEnabled := logger.IsDebug()
	defer logger.SetDebug(debugEnabled)
	if flags.Quiet {
		logger.SetDebug(false)
	}

	command := remaining[0]
	if flags.Describe {
		return fail(flags.JSON, exitConfigError, fmt.Errorf("`-v` cannot be combined with other commands"))
	}
	if providers.Supports(command) {
		return runShort(ctx, command, remaining[1:], flags)
	}
	if canInferProvider(flags) {
		return runInferredProvider(ctx, command, remaining[1:], flags)
	}
	if isHeadlessCommand(command) {
		return fail(flags.JSON, exitConfigError, fmt.Errorf("provider is required unless supplied by --profile, --creds, or --stdin"))
	}
	return fail(flags.JSON, exitConfigError, fmt.Errorf("unsupported command: %s", command))
}

func runShort(ctx context.Context, provider string, args []string, flags commandFlags) int {
	if len(args) == 0 {
		return fail(flags.JSON, exitConfigError, fmt.Errorf("missing payload or action for provider %s", provider))
	}
	payloadName, metadata, err := resolveRunRequest(args[0], args[1:], flags)
	if err != nil {
		return fail(flags.JSON, exitConfigError, err)
	}
	return executeRun(ctx, provider, payloadName, metadata, flags)
}

func runInferredProvider(ctx context.Context, command string, args []string, flags commandFlags) int {
	payloadName, metadata, err := resolveRunRequest(command, args, flags)
	if err != nil {
		return fail(flags.JSON, exitConfigError, err)
	}
	return executeRun(ctx, "", payloadName, metadata, flags)
}

func executeRun(parent context.Context, providerName, payloadName, metadataOverride string, flags commandFlags) int {
	provider := strings.TrimSpace(providerName)
	payloadName = strings.TrimSpace(payloadName)
	payload, resolved, ok := payloads.Lookup(payloadName)
	if !ok {
		return fail(flags.JSON, exitConfigError, fmt.Errorf("unsupported payload: %s", payloadName))
	}
	payloadName = resolved

	config, err := buildRunConfig(provider, payloadName, metadataOverride, flags)
	if err != nil {
		return fail(flags.JSON, exitConfigError, err)
	}
	provider = config[utils.Provider]
	capability := payloads.PayloadCapability(payloadName)
	if capability != "" && !registry.SupportsCapability(provider, capability) {
		return fail(flags.JSON, exitUnsupported, fmt.Errorf("%s does not support %s", provider, payloadName))
	}
	if err := requireApproval(config, flags); err != nil {
		return fail(flags.JSON, exitApprovalRequired, err)
	}

	baseEnv := runner.DefaultEnv()
	if payloadName == "cloudlist" {
		items, err := resolveCloudlistSelection(baseEnv.Cloudlist, metadataOverride)
		if err != nil {
			return fail(flags.JSON, exitConfigError, err)
		}
		baseEnv.Cloudlist = items
	}
	prev := env.Active().Clone()
	env.SetActive(baseEnv)
	defer env.SetActive(prev)

	runCtx, cancel := withRunTimeout(parent, baseEnv.RunTimeout)
	defer cancel()
	ctx := env.With(runCtx, baseEnv)
	result := payloads.Execute(ctx, config, payload)
	exitCode := exitCodeFor(result.Code)
	if flags.JSON {
		if writeCode := writeResultJSON(result, exitCode); writeCode != exitSuccess {
			return writeCode
		}
		return exitCode
	}

	if result.Value != nil {
		if err := payloads.Render(ctx, result.Value); err != nil {
			code := payloads.CodeOutputFailed
			if errors.Is(err, context.DeadlineExceeded) {
				code = payloads.CodeDeadlineExceeded
			} else if errors.Is(err, context.Canceled) {
				code = payloads.CodeCanceled
			}
			return failWithCode(false, exitCodeFor(code), code, err)
		}
	}
	if result.Err != nil {
		return failWithCode(false, exitCode, result.Code, result.Err)
	}
	return exitCode
}

func withRunTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if timeout <= 0 {
		timeout = env.Default().RunTimeout
	}
	return context.WithTimeout(parent, timeout)
}

func exitCodeFor(code payloads.ErrorCode) int {
	switch code {
	case "", payloads.CodeOK:
		return exitSuccess
	case payloads.CodePartialFailure:
		return exitPartial
	case payloads.CodeApprovalRequired, payloads.CodeApprovalRejected:
		return exitApprovalRequired
	case payloads.CodeInvalidArgument:
		return exitConfigError
	case payloads.CodeUnsupported:
		return exitUnsupported
	case payloads.CodeDeadlineExceeded:
		return exitDeadlineExceeded
	case payloads.CodeCanceled:
		return exitCanceled
	default:
		return exitExecutionError
	}
}

func requireApproval(config map[string]string, flags commandFlags) error {
	sensitivity := payloads.DescribeSensitivity(config[utils.Payload], config[utils.Metadata])
	if !sensitivity.RequiresConfirmation() {
		return nil
	}
	if flags.Approval {
		return nil
	}
	if canPromptForApproval(flags) {
		if confirm.Ask(sensitivity.ConfirmKey, config[utils.Provider], sensitivity.Resource) {
			return nil
		}
		return headlessError{
			code:    payloads.CodeApprovalRejected,
			message: "sensitive action was not approved",
		}
	}
	return headlessError{
		code:    payloads.CodeApprovalRequired,
		message: "sensitive action requires -y or --yes",
	}
}

func canInferProvider(flags commandFlags) bool {
	return strings.TrimSpace(flags.Profile) != "" || strings.TrimSpace(flags.CredsPath) != "" || flags.Stdin
}

func canPromptForApproval(flags commandFlags) bool {
	if flags.JSON || flags.Stdin {
		return false
	}
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}
