package headless

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/404tk/cloudtoolkit/pkg/providers"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/runner"
	"github.com/404tk/cloudtoolkit/runner/catalog"
	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/confirm"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func Run(args []string) int {
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
		return runShort(command, remaining[1:], flags)
	}
	if canInferProvider(flags) {
		return runInferredProvider(command, remaining[1:], flags)
	}
	if isHeadlessCommand(command) {
		return fail(flags.JSON, exitConfigError, fmt.Errorf("provider is required unless supplied by --profile, --creds, or --stdin"))
	}
	return fail(flags.JSON, exitConfigError, fmt.Errorf("unsupported command: %s", command))
}

func runShort(provider string, args []string, flags commandFlags) int {
	if len(args) == 0 {
		return fail(flags.JSON, exitConfigError, fmt.Errorf("missing payload or action for provider %s", provider))
	}
	payloadName, metadata, err := resolveRunRequest(args[0], args[1:], flags)
	if err != nil {
		return fail(flags.JSON, exitConfigError, err)
	}
	return executeRun(provider, payloadName, metadata, flags)
}

func runInferredProvider(command string, args []string, flags commandFlags) int {
	payloadName, metadata, err := resolveRunRequest(command, args, flags)
	if err != nil {
		return fail(flags.JSON, exitConfigError, err)
	}
	return executeRun("", payloadName, metadata, flags)
}

func executeRun(providerName, payloadName, metadataOverride string, flags commandFlags) int {
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
	capability := catalog.PayloadCapability(payloadName)
	if capability != "" && !catalog.ProviderSupportsCapability(provider, capability) {
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

	ctx := env.With(context.Background(), baseEnv)
	if !flags.JSON {
		payload.Run(ctx, config)
		return exitSuccess
	}
	producer, ok := payload.(payloads.ResultProducer)
	if !ok {
		return fail(flags.JSON, exitUnsupported, fmt.Errorf("payload %s does not support structured headless output yet; retry without --json", payloadName))
	}

	result, err := producer.Result(ctx, config)
	if err != nil {
		if resultErr, ok := err.(payloads.ResultError); ok {
			if writeCode := writeJSON(resultErr.ResultPayload()); writeCode != exitSuccess {
				return writeCode
			}
			return resultErr.ExitCode()
		}
		return fail(flags.JSON, exitConfigError, err)
	}

	code := exitSuccess
	if cloud, ok := result.(*payloads.CloudListResult); ok && len(cloud.Errors) > 0 {
		code = exitPartial
	}
	if cloud, ok := result.(payloads.CloudListResult); ok && len(cloud.Errors) > 0 {
		code = exitPartial
	}
	if writeCode := writeJSON(result); writeCode != exitSuccess {
		return writeCode
	}
	return code
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
			code:    "approval_rejected",
			message: "sensitive action was not approved",
		}
	}
	return headlessError{
		code:    "approval_required",
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
