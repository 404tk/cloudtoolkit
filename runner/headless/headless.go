package headless

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/404tk/cloudtoolkit/pkg/providers"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/runtime/vmexecspec"
	"github.com/404tk/cloudtoolkit/runner"
	"github.com/404tk/cloudtoolkit/runner/catalog"
	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/confirm"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

const (
	exitSuccess          = 0
	exitPartial          = 2
	exitApprovalRequired = 3
	exitConfigError      = 4
	exitUnsupported      = 5
	schemaVersionV1      = "1"
)

type commandFlags struct {
	JSON      bool
	Quiet     bool
	NoColor   bool
	Agent     bool
	Describe  bool
	Stdin     bool
	Approval  bool
	ShellMode bool
	CmdMode   bool
	Profile   string
	CredsPath string
	Metadata  string

	AccessKey     string
	SecretKey     string
	SecurityToken string
	Region        string
	ProjectID     string
	Version       string
	AzureClientID string
	AzureSecret   string
	AzureTenantID string
	AzureSubID    string
	GCPBase64JSON string
}

type codedError interface {
	error
	ErrorCode() string
}

type headlessError struct {
	code    string
	message string
}

type actionSpec struct {
	payload string
	minArgs int
	maxArgs int
	usage   string
	build   func([]string) string
}

var actionSpecs = map[string]actionSpec{
	"ls": {
		payload: "cloudlist",
		minArgs: 0,
		maxArgs: 1,
		usage:   "ls [resource[,resource...]]",
		build: func(args []string) string {
			if len(args) == 0 {
				return ""
			}
			return strings.TrimSpace(args[0])
		},
	},
	"useradd": {
		payload: "iam-user-check",
		minArgs: 2,
		maxArgs: 2,
		usage:   "useradd <username> <password>",
		build: func(args []string) string {
			return "add " + args[0] + " " + args[1]
		},
	},
	"userdel": {
		payload: "iam-user-check",
		minArgs: 1,
		maxArgs: 1,
		usage:   "userdel <username>",
		build: func(args []string) string {
			return "del " + args[0]
		},
	},
	"bls": {
		payload: "bucket-check",
		minArgs: 0,
		maxArgs: 1,
		usage:   "bls [bucket]",
		build: func(args []string) string {
			if len(args) == 0 {
				return "list all"
			}
			return "list " + args[0]
		},
	},
	"bcnt": {
		payload: "bucket-check",
		minArgs: 0,
		maxArgs: 1,
		usage:   "bcnt [bucket]",
		build: func(args []string) string {
			if len(args) == 0 {
				return "total all"
			}
			return "total " + args[0]
		},
	},
	"shell": {
		payload: "instance-cmd-check",
		minArgs: 2,
		maxArgs: -1,
		usage:   "shell <instance-id> <cmd...> -r <region> (-sh | -cmd)",
	},
}

func (e headlessError) Error() string {
	return e.message
}

func (e headlessError) ErrorCode() string {
	return e.code
}

func Run(args []string) int {
	flags, remaining, err := parseFlags(args)
	if err != nil {
		return fail(flags.JSON, exitConfigError, err)
	}
	if flags.Describe && len(remaining) == 0 {
		return writeVersion(flags.JSON)
	}
	if len(remaining) == 0 {
		return fail(flags.JSON, exitConfigError, errors.New("missing command"))
	}

	logger.SetOutputs(os.Stderr, os.Stderr)
	defer logger.SetOutputs(os.Stdout, os.Stderr)
	processbar.SetOutput(os.Stderr)
	defer processbar.SetOutput(nil)
	debugEnabled := logger.IsDebug()
	defer logger.SetDebug(debugEnabled)
	if flags.Quiet || flags.Agent {
		logger.SetDebug(false)
	}

	command := remaining[0]
	if flags.Describe {
		return fail(flags.JSON, exitConfigError, errors.New("`-v` cannot be combined with other commands"))
	}
	if command == "describe" {
		return fail(flags.JSON, exitConfigError, fmt.Errorf("unsupported: %s", command))
	}
	if providers.Supports(command) {
		return runShort(command, remaining[1:], flags)
	}
	if canInferProvider(flags) {
		return runInferredProvider(command, remaining[1:], flags)
	}
	if isHeadlessCommand(command) {
		return fail(flags.JSON, exitConfigError, errors.New("provider is required unless supplied by --profile, --creds, or --stdin"))
	}
	return fail(flags.JSON, exitConfigError, fmt.Errorf("unsupported command: %s", command))
}

func parseFlags(args []string) (commandFlags, []string, error) {
	var cfg commandFlags
	fs := newFlagSet("ctk", &cfg)
	normalized, err := normalizeArgs(args)
	if err != nil {
		return commandFlags{}, nil, err
	}
	if err := fs.Parse(normalized); err != nil {
		return commandFlags{}, nil, err
	}
	cfg.applyDerived()
	return cfg, fs.Args(), nil
}

func newFlagSet(name string, cfg *commandFlags) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.BoolVar(&cfg.JSON, "json", cfg.JSON, "emit JSON")
	fs.BoolVar(&cfg.Quiet, "quiet", cfg.Quiet, "reduce log chatter")
	fs.BoolVar(&cfg.NoColor, "no-color", cfg.NoColor, "disable ANSI color")
	fs.BoolVar(&cfg.Agent, "agent", cfg.Agent, "agent-friendly mode")
	fs.BoolVar(&cfg.Describe, "v", cfg.Describe, "print version")
	fs.BoolVar(&cfg.Stdin, "stdin", cfg.Stdin, "read credentials JSON from stdin")
	fs.BoolVar(&cfg.Approval, "yes", cfg.Approval, "approve sensitive execution")
	fs.BoolVar(&cfg.Approval, "y", cfg.Approval, "approve sensitive execution")
	fs.BoolVar(&cfg.ShellMode, "sh", cfg.ShellMode, "")
	fs.BoolVar(&cfg.CmdMode, "cmd", cfg.CmdMode, "")
	fs.StringVar(&cfg.Profile, "profile", cfg.Profile, "credential profile name")
	fs.StringVar(&cfg.Profile, "P", cfg.Profile, "credential profile name")
	fs.StringVar(&cfg.CredsPath, "creds", cfg.CredsPath, "credentials JSON file")
	fs.StringVar(&cfg.Metadata, "metadata", cfg.Metadata, "payload metadata")

	fs.StringVar(&cfg.AccessKey, "accesskey", cfg.AccessKey, "")
	fs.StringVar(&cfg.AccessKey, "k", cfg.AccessKey, "")
	fs.StringVar(&cfg.SecretKey, "secretkey", cfg.SecretKey, "")
	fs.StringVar(&cfg.SecretKey, "s", cfg.SecretKey, "")
	fs.StringVar(&cfg.SecurityToken, "token", cfg.SecurityToken, "")
	fs.StringVar(&cfg.Region, "region", cfg.Region, "")
	fs.StringVar(&cfg.Region, "r", cfg.Region, "")
	fs.StringVar(&cfg.ProjectID, "projectId", cfg.ProjectID, "")
	fs.StringVar(&cfg.Version, "version", cfg.Version, "")
	fs.StringVar(&cfg.AzureClientID, "clientId", cfg.AzureClientID, "")
	fs.StringVar(&cfg.AzureSecret, "clientSecret", cfg.AzureSecret, "")
	fs.StringVar(&cfg.AzureTenantID, "tenantId", cfg.AzureTenantID, "")
	fs.StringVar(&cfg.AzureSubID, "subscriptionId", cfg.AzureSubID, "")
	fs.StringVar(&cfg.GCPBase64JSON, "base64Json", cfg.GCPBase64JSON, "")
	return fs
}

func (cfg *commandFlags) applyDerived() {
	if cfg.Agent {
		cfg.JSON = true
		cfg.Quiet = true
		cfg.NoColor = true
	}
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

func buildRunConfig(provider, payload, metadataOverride string, flags commandFlags) (map[string]string, error) {
	sourceData, err := credentialDataFromFlags(flags)
	if err != nil {
		return nil, err
	}
	sourceProvider := strings.TrimSpace(sourceData[utils.Provider])
	resolvedProvider := strings.TrimSpace(provider)
	if resolvedProvider == "" {
		resolvedProvider = sourceProvider
	}
	if resolvedProvider == "" {
		return nil, errors.New("provider is required unless supplied by the selected credential source")
	}
	if sourceProvider != "" && sourceProvider != resolvedProvider {
		return nil, fmt.Errorf("provider mismatch: command selected %s but credential source is for %s", resolvedProvider, sourceProvider)
	}
	if _, ok := catalog.ProviderSpecFor(resolvedProvider); !ok {
		return nil, fmt.Errorf("unsupported provider: %s", resolvedProvider)
	}
	config, _ := catalog.DefaultProviderConfig(resolvedProvider)
	config[utils.Provider] = resolvedProvider
	config[utils.Payload] = payload
	config[utils.Metadata] = strings.TrimSpace(metadataOverride)

	mergeConfig(config, sourceData)
	mergeConfig(config, flags.explicitProviderOptions())

	config[utils.Provider] = resolvedProvider
	config[utils.Payload] = payload
	if strings.TrimSpace(metadataOverride) != "" {
		config[utils.Metadata] = strings.TrimSpace(metadataOverride)
	} else if strings.TrimSpace(flags.Metadata) != "" {
		config[utils.Metadata] = strings.TrimSpace(flags.Metadata)
	}
	return config, nil
}

func credentialDataFromFlags(flags commandFlags) (map[string]string, error) {
	sourceCount := 0
	if strings.TrimSpace(flags.Profile) != "" {
		sourceCount++
	}
	if strings.TrimSpace(flags.CredsPath) != "" {
		sourceCount++
	}
	if flags.Stdin {
		sourceCount++
	}
	if sourceCount > 1 {
		return nil, errors.New("credential sources are mutually exclusive: choose one of --profile, --creds, or --stdin")
	}

	switch {
	case strings.TrimSpace(flags.Profile) != "":
		return loadProfile(flags.Profile)
	case strings.TrimSpace(flags.CredsPath) != "":
		return loadCredentialFile(flags.CredsPath)
	case flags.Stdin:
		return loadCredentialStdin()
	default:
		return nil, nil
	}
}

func (flags commandFlags) explicitProviderOptions() map[string]string {
	items := map[string]string{}
	if flags.AccessKey != "" {
		items[utils.AccessKey] = flags.AccessKey
	}
	if flags.SecretKey != "" {
		items[utils.SecretKey] = flags.SecretKey
	}
	if flags.SecurityToken != "" {
		items[utils.SecurityToken] = flags.SecurityToken
	}
	if flags.Region != "" {
		items[utils.Region] = flags.Region
	}
	if flags.ProjectID != "" {
		items[utils.ProjectID] = flags.ProjectID
	}
	if flags.Version != "" {
		items[utils.Version] = flags.Version
	}
	if flags.AzureClientID != "" {
		items[utils.AzureClientId] = flags.AzureClientID
	}
	if flags.AzureSecret != "" {
		items[utils.AzureClientSecret] = flags.AzureSecret
	}
	if flags.AzureTenantID != "" {
		items[utils.AzureTenantId] = flags.AzureTenantID
	}
	if flags.AzureSubID != "" {
		items[utils.AzureSubscriptionId] = flags.AzureSubID
	}
	if flags.GCPBase64JSON != "" {
		items[utils.GCPserviceAccountJSON] = flags.GCPBase64JSON
	}
	return items
}

func writeJSON(v any) int {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return exitConfigError
	}
	if _, err := os.Stdout.Write(append(data, '\n')); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return exitConfigError
	}
	return exitSuccess
}

func writeVersion(jsonOutput bool) int {
	if jsonOutput {
		return writeJSON(map[string]string{
			"version": runner.Version(),
		})
	}
	if _, err := fmt.Fprintf(os.Stdout, "CloudToolKit %s\n", runner.Version()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return exitConfigError
	}
	return exitSuccess
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

func fail(jsonOutput bool, code int, err error) int {
	if err == nil {
		return code
	}
	if jsonOutput {
		payload := map[string]any{
			"schema_version": schemaVersionV1,
			"error":          err.Error(),
			"exit_code":      code,
		}
		if coded, ok := err.(codedError); ok {
			payload["code"] = coded.ErrorCode()
		}
		_ = writeJSON(payload)
		return code
	}
	fmt.Fprintln(os.Stderr, err.Error())
	return code
}

func mergeConfig(dst, src map[string]string) {
	for key, value := range src {
		if strings.TrimSpace(value) == "" {
			continue
		}
		dst[key] = value
	}
}

func loadCredentialFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeCredentialJSON(data)
}

func loadCredentialStdin() (map[string]string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	return decodeCredentialJSON(data)
}

func loadProfile(profile string) (map[string]string, error) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return nil, errors.New("empty profile name")
	}

	if id, err := findProfileID(profile); err == nil {
		return decodeSessionJSON(cache.Cfg.CredSelect(id))
	}
	return decodeSessionJSON(cache.Cfg.CredSelect(profile))
}

func findProfileID(profile string) (string, error) {
	for _, cred := range cache.Cfg.Snapshot() {
		if cred.Note == profile || cred.UUID == profile {
			return cred.UUID, nil
		}
	}
	return "", fmt.Errorf("profile not found: %s", profile)
}

func decodeSessionJSON(data string) (map[string]string, error) {
	if strings.TrimSpace(data) == "" {
		return nil, errors.New("empty cached session")
	}
	return decodeCredentialJSON([]byte(data))
}

func decodeCredentialJSON(data []byte) (map[string]string, error) {
	items := make(map[string]string)
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func resolveRunRequest(command string, args []string, flags commandFlags) (string, string, error) {
	command = strings.TrimSpace(command)
	if command == "shell" {
		return resolveShellAction(args, flags)
	}
	if spec, ok := actionSpecs[command]; ok {
		if strings.TrimSpace(flags.Metadata) != "" {
			return "", "", fmt.Errorf("%s does not accept --metadata; use `%s`", command, spec.usage)
		}
		if len(args) < spec.minArgs {
			return "", "", fmt.Errorf("usage: %s", spec.usage)
		}
		if spec.maxArgs >= 0 && len(args) > spec.maxArgs {
			return "", "", fmt.Errorf("usage: %s", spec.usage)
		}
		if spec.build == nil {
			return "", "", fmt.Errorf("usage: %s", spec.usage)
		}
		return spec.payload, spec.build(args), nil
	}
	return "", "", fmt.Errorf("unsupported: %s", command)
}

func resolveShellAction(args []string, flags commandFlags) (string, string, error) {
	if strings.TrimSpace(flags.Metadata) != "" {
		return "", "", errors.New("shell does not accept --metadata")
	}
	if len(args) < 2 {
		return "", "", errors.New("usage: shell <instance-id> <cmd...> -r <region> (-sh | -cmd)")
	}
	if strings.TrimSpace(flags.Region) == "" || strings.EqualFold(strings.TrimSpace(flags.Region), "all") {
		return "", "", errors.New("headless shell requires explicit -r <region>; region=all is not supported")
	}
	if flags.ShellMode == flags.CmdMode {
		return "", "", errors.New("headless shell requires exactly one of -sh or -cmd")
	}

	instanceID := strings.TrimSpace(args[0])
	command := strings.Join(args[1:], " ")
	if instanceID == "" || strings.TrimSpace(command) == "" {
		return "", "", errors.New("usage: shell <instance-id> <cmd...> -r <region> (-sh | -cmd)")
	}

	if flags.ShellMode {
		return "instance-cmd-check", instanceID + " " + vmexecspec.BuildLinux(command), nil
	}
	return "instance-cmd-check", instanceID + " " + vmexecspec.BuildWindows(command), nil
}

func isHeadlessCommand(command string) bool {
	if _, ok := actionSpecs[strings.TrimSpace(command)]; ok {
		return true
	}
	return false
}

func canInferProvider(flags commandFlags) bool {
	return strings.TrimSpace(flags.Profile) != "" || strings.TrimSpace(flags.CredsPath) != "" || flags.Stdin
}

func canPromptForApproval(flags commandFlags) bool {
	if flags.JSON || flags.Agent || flags.Stdin {
		return false
	}
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}

func normalizeArgs(args []string) ([]string, error) {
	flagArgs := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	sawShell := false
	shellTargetSeen := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			if arg == "shell" && !sawShell {
				sawShell = true
				shellTargetSeen = false
				continue
			}
			if sawShell && !shellTargetSeen {
				shellTargetSeen = true
			}
			continue
		}

		name, hasValue := parseFlagToken(arg)
		if sawShell && shellTargetSeen && !isBoolFlag(name) && !isValueFlag(name) {
			positionals = append(positionals, arg)
			continue
		}
		switch {
		case isBoolFlag(name):
			flagArgs = append(flagArgs, arg)
		case isValueFlag(name):
			flagArgs = append(flagArgs, arg)
			if !hasValue {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("flag needs an argument: -%s", name)
				}
				i++
				flagArgs = append(flagArgs, args[i])
			}
		default:
			return nil, fmt.Errorf("flag provided but not defined: %s", arg)
		}
	}
	return append(flagArgs, positionals...), nil
}

func resolveCloudlistSelection(defaults []string, selection string) ([]string, error) {
	selection = strings.TrimSpace(selection)
	if selection == "" || strings.EqualFold(selection, "all") {
		return append([]string(nil), defaults...), nil
	}

	aliases := map[string]string{
		"all":      "all",
		"balance":  "balance",
		"amt":      "balance",
		"host":     "host",
		"vm":       "host",
		"user":     "account",
		"account":  "account",
		"iam":      "account",
		"bucket":   "bucket",
		"s3":       "bucket",
		"database": "database",
		"db":       "database",
		"rds":      "database",
		"domain":   "domain",
		"dns":      "domain",
		"sms":      "sms",
		"log":      "log",
		"sls":      "log",
	}

	items := make([]string, 0)
	seen := make(map[string]struct{})
	for _, raw := range strings.Split(selection, ",") {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		mapped, ok := aliases[key]
		if !ok {
			return nil, fmt.Errorf("unknown cloudlist resource type: %s", raw)
		}
		if mapped == "all" {
			return append([]string(nil), defaults...), nil
		}
		if _, ok := seen[mapped]; ok {
			continue
		}
		seen[mapped] = struct{}{}
		items = append(items, mapped)
	}
	if len(items) == 0 {
		return append([]string(nil), defaults...), nil
	}
	return items, nil
}

func parseFlagToken(arg string) (name string, hasValue bool) {
	trimmed := strings.TrimLeft(arg, "-")
	parts := strings.SplitN(trimmed, "=", 2)
	return parts[0], len(parts) == 2
}

func isBoolFlag(name string) bool {
	switch name {
	case "json", "quiet", "no-color", "agent", "stdin", "v", "yes", "y", "sh", "cmd":
		return true
	default:
		return false
	}
}

func isValueFlag(name string) bool {
	switch name {
	case "profile", "P", "creds", "metadata", "accesskey", "k", "secretkey", "s", "token", "region", "r", "projectId", "version", "clientId", "clientSecret", "tenantId", "subscriptionId", "base64Json":
		return true
	default:
		return false
	}
}
