package headless

import (
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/404tk/cloudtoolkit/runner/catalog"
	"github.com/404tk/cloudtoolkit/utils"
)

type providerFlagBinding struct {
	long      string
	short     string
	aliases   []string
	valueName string
	help      string
}

type providerOptionValue struct {
	cfg *commandFlags
	key string
}

func (v *providerOptionValue) String() string {
	if v == nil || v.cfg == nil {
		return ""
	}
	return v.cfg.providerOption(v.key)
}

func (v *providerOptionValue) Set(value string) error {
	if v == nil || v.cfg == nil {
		return nil
	}
	v.cfg.setProviderOption(v.key, value)
	return nil
}

var providerFlagBindings = map[string]providerFlagBinding{
	utils.AccessKey: {
		long:      utils.AccessKey,
		short:     "ak",
		aliases:   []string{"k"},
		valueName: "key",
		help:      "provider access key",
	},
	utils.SecretKey: {
		long:      utils.SecretKey,
		short:     "sk",
		aliases:   []string{"s"},
		valueName: "secret",
		help:      "provider secret key",
	},
	utils.SecurityToken: {
		long:      utils.SecurityToken,
		short:     "st",
		valueName: "token",
		help:      "provider security token",
	},
	utils.Region: {
		long:      utils.Region,
		short:     "r",
		valueName: "region",
		help:      "provider region",
	},
	utils.ProjectID: {
		long:      utils.ProjectID,
		valueName: "id",
		help:      "UCloud project ID",
	},
	utils.Version: {
		long:      utils.Version,
		valueName: "value",
		help:      "provider version or edition",
	},
	utils.AzureClientId: {
		long:      utils.AzureClientId,
		valueName: "id",
		help:      "Azure client ID",
	},
	utils.AzureClientSecret: {
		long:      utils.AzureClientSecret,
		valueName: "secret",
		help:      "Azure client secret",
	},
	utils.AzureTenantId: {
		long:      utils.AzureTenantId,
		valueName: "id",
		help:      "Azure tenant ID",
	},
	utils.AzureSubscriptionId: {
		long:      utils.AzureSubscriptionId,
		valueName: "id",
		help:      "Azure subscription ID",
	},
	utils.GCPserviceAccountJSON: {
		long:      utils.GCPserviceAccountJSON,
		valueName: "value",
		help:      "Base64-encoded GCP service account JSON",
	},
}

var providerFlagBindingOrder = []string{
	utils.AccessKey,
	utils.SecretKey,
	utils.SecurityToken,
	utils.Region,
	utils.ProjectID,
	utils.Version,
	utils.AzureClientId,
	utils.AzureClientSecret,
	utils.AzureTenantId,
	utils.AzureSubscriptionId,
	utils.GCPserviceAccountJSON,
}

var commonHeadlessFlagSpecs = []headlessFlagSpec{
	{
		long:    "json",
		kind:    flagBool,
		help:    "emit JSON",
		section: helpCommon,
		bind: func(fs *flag.FlagSet, cfg *commandFlags) {
			fs.BoolVar(&cfg.JSON, "json", cfg.JSON, "emit JSON")
		},
	},
	{
		long:    "quiet",
		kind:    flagBool,
		help:    "reduce log chatter",
		section: helpCommon,
		bind: func(fs *flag.FlagSet, cfg *commandFlags) {
			fs.BoolVar(&cfg.Quiet, "quiet", cfg.Quiet, "reduce log chatter")
		},
	},
	{
		short:   "v",
		kind:    flagBool,
		section: helpHidden,
		bind: func(fs *flag.FlagSet, cfg *commandFlags) {
			fs.BoolVar(&cfg.Describe, "v", cfg.Describe, "print version")
		},
	},
	{
		long:    "stdin",
		kind:    flagBool,
		help:    "read credentials JSON from stdin",
		section: helpCommon,
		bind: func(fs *flag.FlagSet, cfg *commandFlags) {
			fs.BoolVar(&cfg.Stdin, "stdin", cfg.Stdin, "read credentials JSON from stdin")
		},
	},
	{
		long:    "yes",
		short:   "y",
		kind:    flagBool,
		help:    "approve sensitive actions",
		section: helpCommon,
		bind: func(fs *flag.FlagSet, cfg *commandFlags) {
			fs.BoolVar(&cfg.Approval, "yes", cfg.Approval, "approve sensitive execution")
			fs.BoolVar(&cfg.Approval, "y", cfg.Approval, "approve sensitive execution")
		},
	},
	{
		short:   "sh",
		kind:    flagBool,
		section: helpHidden,
		bind: func(fs *flag.FlagSet, cfg *commandFlags) {
			fs.BoolVar(&cfg.ShellMode, "sh", cfg.ShellMode, "")
		},
	},
	{
		short:   "cmd",
		kind:    flagBool,
		section: helpHidden,
		bind: func(fs *flag.FlagSet, cfg *commandFlags) {
			fs.BoolVar(&cfg.CmdMode, "cmd", cfg.CmdMode, "")
		},
	},
	{
		long:      "profile",
		short:     "P",
		kind:      flagValue,
		valueName: "name",
		help:      "use cached credential profile",
		section:   helpCommon,
		bind: func(fs *flag.FlagSet, cfg *commandFlags) {
			fs.StringVar(&cfg.Profile, "profile", cfg.Profile, "credential profile name")
			fs.StringVar(&cfg.Profile, "P", cfg.Profile, "credential profile name")
		},
	},
	{
		long:      "creds",
		kind:      flagValue,
		valueName: "file",
		help:      "read credentials JSON file",
		section:   helpCommon,
		bind: func(fs *flag.FlagSet, cfg *commandFlags) {
			fs.StringVar(&cfg.CredsPath, "creds", cfg.CredsPath, "credentials JSON file")
		},
	},
	{
		long:      "metadata",
		kind:      flagValue,
		valueName: "value",
		section:   helpHidden,
		bind: func(fs *flag.FlagSet, cfg *commandFlags) {
			fs.StringVar(&cfg.Metadata, "metadata", cfg.Metadata, "payload metadata")
		},
	},
}

var headlessFlagSpecs = buildHeadlessFlagSpecs()
var boolFlagNames = buildFlagNames(flagBool)
var valueFlagNames = buildFlagNames(flagValue)

func parseFlags(args []string) (commandFlags, []string, error) {
	cfg := commandFlags{
		providerValues: make(map[string]string),
	}
	fs := newFlagSet("ctk", &cfg)
	normalized, err := normalizeArgs(args)
	if err != nil {
		return commandFlags{}, nil, err
	}
	if err := fs.Parse(normalized); err != nil {
		return commandFlags{}, nil, err
	}
	return cfg, fs.Args(), nil
}

func newFlagSet(name string, cfg *commandFlags) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	for _, spec := range headlessFlagSpecs {
		spec.bind(fs, cfg)
	}
	return fs
}

func wantsHelp(args []string) bool {
	sawShell := false
	shellTargetSeen := false
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
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
		if sawShell && shellTargetSeen {
			continue
		}
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
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

func parseFlagToken(arg string) (name string, hasValue bool) {
	trimmed := strings.TrimLeft(arg, "-")
	parts := strings.SplitN(trimmed, "=", 2)
	return parts[0], len(parts) == 2
}

func isBoolFlag(name string) bool {
	_, ok := boolFlagNames[name]
	return ok
}

func isValueFlag(name string) bool {
	_, ok := valueFlagNames[name]
	return ok
}

func buildFlagNames(kind flagKind) map[string]struct{} {
	names := make(map[string]struct{})
	for _, spec := range headlessFlagSpecs {
		if spec.kind != kind {
			continue
		}
		for _, name := range rawFlagNames(spec) {
			names[name] = struct{}{}
		}
	}
	return names
}

func buildHeadlessFlagSpecs() []headlessFlagSpec {
	specs := make([]headlessFlagSpec, 0, len(commonHeadlessFlagSpecs)+len(providerFlagBindings))
	specs = append(specs, commonHeadlessFlagSpecs...)
	specs = append(specs, providerOptionFlagSpecs()...)
	return specs
}

func providerOptionFlagSpecs() []headlessFlagSpec {
	specs := make([]headlessFlagSpec, 0)
	for _, optionName := range orderedProviderOptionNames() {
		binding := providerFlagBindings[optionName]
		optionName := optionName
		long := firstNonEmpty(binding.long, optionName)
		short := binding.short
		aliases := append([]string(nil), binding.aliases...)
		valueName := firstNonEmpty(binding.valueName, "value")
		help := firstNonEmpty(binding.help, catalog.OptionDescription(optionName))
		specs = append(specs, headlessFlagSpec{
			long:      long,
			short:     short,
			aliases:   aliases,
			kind:      flagValue,
			valueName: valueName,
			help:      help,
			section:   helpProvider,
			bind: func(fs *flag.FlagSet, cfg *commandFlags) {
				bindProviderOptionFlag(fs, cfg, optionName, long, short, aliases...)
			},
		})
	}
	return specs
}

func orderedProviderOptionNames() []string {
	available := make(map[string]struct{})
	for _, name := range catalog.OptionNames() {
		if name == utils.Payload || name == utils.Metadata {
			continue
		}
		available[name] = struct{}{}
	}

	ordered := make([]string, 0, len(available))
	for _, name := range providerFlagBindingOrder {
		if _, ok := available[name]; !ok {
			continue
		}
		ordered = append(ordered, name)
		delete(available, name)
	}

	extras := make([]string, 0, len(available))
	for name := range available {
		extras = append(extras, name)
	}
	sort.Strings(extras)
	return append(ordered, extras...)
}

func bindProviderOptionFlag(fs *flag.FlagSet, cfg *commandFlags, optionName, long, short string, aliases ...string) {
	value := &providerOptionValue{cfg: cfg, key: optionName}
	seen := make(map[string]struct{})
	names := make([]string, 0, 2+len(aliases))
	names = append(names, long, short)
	names = append(names, aliases...)
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		fs.Var(value, name, "")
	}
}

func rawFlagNames(spec headlessFlagSpec) []string {
	names := make([]string, 0, 2+len(spec.aliases))
	if spec.long != "" {
		names = append(names, spec.long)
	}
	if spec.short != "" {
		names = append(names, spec.short)
	}
	names = append(names, spec.aliases...)
	return names
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
