package console

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/go-prompt"
)

type CompletionContext struct {
	Mode       HelpMode
	Provider   string
	Payload    string
	Config     map[string]string
	InstanceID string
	DemoReplay bool
}

type shellTargetHint struct {
	Provider string
	Source   string
}

var knownShellTargets = make(map[string]shellTargetHint)

var commandSuggestionDescriptions = map[string]string{
	"help":     "show context-aware help",
	"use":      "enter provider mode",
	"sessions": "list or reuse sessions",
	"note":     "annotate a session",
	"clear":    "clear the current screen",
	"exit":     "leave the current mode",
	// "quit":     "leave the current mode",
	// "back":     "return to the previous mode",
	"show":  "show options or payloads",
	"set":   "set an option or payload parameter",
	"demo":  "enable deterministic replay for supported providers",
	"run":   "run the selected payload",
	"shell": "open an authorized instance shell",
}

var rootCommandNames = []string{
	"help",
	"use",
	"sessions",
	"note",
	"clear",
	"exit",
}

var shellCommandNames = []string{
	"help",
	"clear",
	"exit",
}

var sessionsCommandSuggestions = []prompt.Suggest{
	{Text: "-i", Description: "interact with a cached session by ID"},
	{Text: "-k", Description: "delete a cached session by ID"},
	{Text: "-c", Description: "check one cached session or all sessions"},
}

var providerCommandNames = []string{
	"help",
	"show",
	"set",
	"demo",
	"run",
	"shell",
	"sessions",
	"note",
	"use",
	"clear",
	"exit",
}

var showTopicSuggestionsData = []prompt.Suggest{
	{Text: "options", Description: "display provider configuration"},
	{Text: "payloads", Description: "display visible validation payloads"},
}

var versionSuggestionsData = []prompt.Suggest{
	{Text: "Intl", Description: "international edition"},
	{Text: "China", Description: "china edition"},
}

var regionSuggestionsByProvider = map[string][]prompt.Suggest{
	"aws": {
		{Text: "all", Description: "enumerate all configured regions"},
		{Text: "us-east-1", Description: "N. Virginia"},
		{Text: "us-east-2", Description: "Ohio"},
		{Text: "us-west-1", Description: "N. California"},
		{Text: "us-west-2", Description: "Oregon"},
		{Text: "ap-east-1", Description: "Hong Kong"},
		{Text: "ap-southeast-1", Description: "Singapore"},
		{Text: "ap-southeast-2", Description: "Sydney"},
		{Text: "ap-northeast-1", Description: "Tokyo"},
		{Text: "ap-northeast-2", Description: "Seoul"},
		{Text: "eu-west-1", Description: "Ireland"},
		{Text: "eu-central-1", Description: "Frankfurt"},
	},
	"alibaba": {
		{Text: "all", Description: "enumerate all configured regions"},
		{Text: "cn-beijing", Description: "Beijing"},
		{Text: "cn-hangzhou", Description: "Hangzhou"},
		{Text: "cn-shanghai", Description: "Shanghai"},
		{Text: "cn-shenzhen", Description: "Shenzhen"},
		{Text: "cn-hongkong", Description: "Hong Kong"},
		{Text: "ap-southeast-1", Description: "Singapore"},
		{Text: "us-east-1", Description: "Virginia"},
		{Text: "eu-central-1", Description: "Frankfurt"},
	},
	"tencent": {
		{Text: "all", Description: "enumerate all configured regions"},
		{Text: "ap-beijing", Description: "Beijing"},
		{Text: "ap-shanghai", Description: "Shanghai"},
		{Text: "ap-guangzhou", Description: "Guangzhou"},
		{Text: "ap-hongkong", Description: "Hong Kong"},
		{Text: "ap-singapore", Description: "Singapore"},
		{Text: "ap-seoul", Description: "Seoul"},
		{Text: "ap-tokyo", Description: "Tokyo"},
	},
	"huawei": {
		{Text: "all", Description: "enumerate all configured regions"},
		{Text: "cn-north-4", Description: "Beijing 4"},
		{Text: "cn-east-3", Description: "Shanghai 1"},
		{Text: "cn-south-1", Description: "Guangzhou"},
		{Text: "ap-southeast-1", Description: "Hong Kong"},
		{Text: "eu-west-101", Description: "Dublin"},
	},
	"volcengine": {
		{Text: "all", Description: "enumerate all configured regions"},
		{Text: "cn-beijing", Description: "Beijing"},
		{Text: "cn-shanghai", Description: "Shanghai"},
		{Text: "ap-southeast-1", Description: "Singapore"},
	},
	"jdcloud": {
		{Text: "all", Description: "enumerate all configured regions"},
		{Text: "cn-north-1", Description: "Beijing"},
		{Text: "cn-east-2", Description: "Shanghai"},
		{Text: "cn-east-1", Description: "Suqian"},
		{Text: "cn-south-1", Description: "Guangzhou"},
	},
	"ucloud": {
		{Text: "all", Description: "enumerate all accessible regions"},
		{Text: "cn-bj2", Description: "Beijing"},
		{Text: "cn-sh2", Description: "Shanghai"},
		{Text: "cn-gd", Description: "Guangzhou"},
		{Text: "hk", Description: "Hong Kong"},
		{Text: "sg", Description: "Singapore"},
		{Text: "us-ca", Description: "Los Angeles"},
		{Text: "th-bkk", Description: "Bangkok"},
		{Text: "ge-fra", Description: "Frankfurt"},
	},
}

var metadataTemplatesByPayload = map[string][]prompt.Suggest{
	"iam-user-check": {
		{Text: "add <username> <password>", Description: "create a validation IAM user"},
		{Text: "del <username>", Description: "remove a validation IAM user"},
	},
	"bucket-check": {
		{Text: "list <bucket-name>", Description: "review bucket contents in an authorized environment"},
		{Text: "total <bucket-name>", Description: "count objects in a bucket"},
	},
	"event-check": {
		{Text: "dump all", Description: "review all relevant events"},
		{Text: "dump <source-ip>", Description: "review events for one source IP"},
	},
	"rds-account-check": {
		{Text: "useradd <instance-id>", Description: "provision a validation database account"},
	},
	"instance-cmd-check": {
		{Text: "<instance-id> <cmd>", Description: "direct metadata syntax for one remote command; prefer `shell <instance-id>` for interactive use"},
	},
}

func currentCompletionContext() CompletionContext {
	helpCtx := currentHelpContext()
	return CompletionContext{
		Mode:       helpCtx.Mode,
		Provider:   helpCtx.Provider,
		Payload:    helpCtx.Payload,
		Config:     config,
		InstanceID: helpCtx.InstanceID,
		DemoReplay: helpCtx.DemoReplay,
	}
}

func completionContextForMode(mode HelpMode) CompletionContext {
	ctx := currentCompletionContext()
	ctx.Mode = mode
	switch mode {
	case HelpModeRoot:
		ctx.Provider = ""
		ctx.Payload = ""
		ctx.InstanceID = ""
		ctx.DemoReplay = false
	case HelpModeShell:
		if ctx.Payload == "" {
			ctx.Payload = "instance-cmd-check"
		}
	}
	return ctx
}

func buildCommandSuggestions(commands []string) []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0, len(commands))
	for _, command := range commands {
		desc := commandSuggestionDescriptions[command]
		suggestions = append(suggestions, prompt.Suggest{
			Text:        command,
			Description: desc,
		})
	}
	return suggestions
}

func commandNamesForContext(mode HelpMode, demoReplay bool, provider string) []string {
	switch mode {
	case HelpModeProvider:
		return providerCommandNamesForState(demoReplay, provider)
	case HelpModeShell:
		return append([]string(nil), shellCommandNames...)
	default:
		return append([]string(nil), rootCommandNames...)
	}
}

func providerCommandNamesForState(demoReplay bool, provider string) []string {
	names := make([]string, 0, len(providerCommandNames))
	demoSupported := demoreplay.SupportsProvider(provider)
	for _, name := range providerCommandNames {
		switch name {
		case "demo":
			if demoReplay || !demoSupported {
				continue
			}
		case "note", "use":
			if demoReplay {
				continue
			}
		}
		names = append(names, name)
	}
	return names
}

func commandSuggestionsForContext(ctx CompletionContext) []prompt.Suggest {
	return buildCommandSuggestions(commandNamesForContext(ctx.Mode, ctx.DemoReplay, ctx.Provider))
}

func commandAvailable(mode HelpMode, demoReplay bool, provider, command string) bool {
	for _, name := range commandNamesForContext(mode, demoReplay, provider) {
		if name == command {
			return true
		}
	}
	return false
}

func noteSuggestions(args []string, word string) []prompt.Suggest {
	if len(args) == 2 {
		return prompt.FilterContains(sessionIDSuggestions(), word, true)
	}
	return []prompt.Suggest{}
}

func sessionsSuggestions(args []string, word string) []prompt.Suggest {
	if len(args) == 2 {
		return prompt.FilterContains(sessionsCommandSuggestions, word, true)
	}
	if len(args) != 3 {
		return []prompt.Suggest{}
	}

	switch args[1] {
	case "-i", "use", "enter", "internation", "interact", "-k", "delete", "kill":
		return prompt.FilterContains(sessionIDSuggestions(), word, true)
	case "-c", "check":
		suggestions := append([]prompt.Suggest{{Text: "all", Description: "validate every cached session"}}, sessionIDSuggestions()...)
		return prompt.FilterContains(suggestions, word, true)
	}
	return []prompt.Suggest{}
}

func sessionIDSuggestions() []prompt.Suggest {
	loadCred()
	suggestions := make([]prompt.Suggest, 0, len(creds))
	for _, cred := range creds {
		desc := fmt.Sprintf("%s / %s", cred.Provider, cred.User)
		if strings.TrimSpace(cred.Note) != "" {
			desc = fmt.Sprintf("%s / %s", desc, cred.Note)
		}
		suggestions = append(suggestions, prompt.Suggest{
			Text:        strconv.Itoa(cred.Id),
			Description: desc,
		})
	}
	return suggestions
}

func getProviderRegionSuggestions(provider string) []prompt.Suggest {
	return regionSuggestionsByProvider[provider]
}

func getPayloadMetadataSuggestions(payload string) []prompt.Suggest {
	return metadataTemplatesByPayload[payloads.ResolveName(payload)]
}

func getShellTargetSuggestions(ctx CompletionContext) []prompt.Suggest {
	candidates := make(map[string]shellTargetHint)
	add := func(target, provider, source string) {
		target = strings.TrimSpace(target)
		if target == "" {
			return
		}
		hint := candidates[target]
		if hint.Provider == "" {
			hint.Provider = strings.TrimSpace(provider)
		}
		if hint.Source == "" {
			hint.Source = strings.TrimSpace(source)
		}
		candidates[target] = hint
	}

	for target, hint := range knownShellTargets {
		add(target, hint.Provider, hint.Source)
	}
	if ctx.InstanceID != "" {
		add(ctx.InstanceID, ctx.Provider, "current shell target")
	}
	if target := shellTargetFromConfig(ctx.Config); target != "" {
		add(target, ctx.Provider, "current config")
	}

	for _, cred := range cache.Cfg.Snapshot() {
		m := make(map[string]string)
		if err := json.Unmarshal([]byte(cred.JsonData), &m); err != nil {
			continue
		}
		if target := shellTargetFromConfig(m); target != "" {
			add(target, m[utils.Provider], "cached session")
		}
	}

	keys := make([]string, 0, len(candidates))
	for key := range candidates {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	suggestions := make([]prompt.Suggest, 0, len(keys))
	for _, key := range keys {
		hint := candidates[key]
		descParts := make([]string, 0, 2)
		if hint.Provider != "" {
			descParts = append(descParts, hint.Provider)
		}
		if hint.Source != "" {
			descParts = append(descParts, hint.Source)
		}
		suggestions = append(suggestions, prompt.Suggest{
			Text:        key,
			Description: strings.Join(descParts, " / "),
		})
	}
	return suggestions
}

func shellTargetFromConfig(cfg map[string]string) string {
	if cfg == nil {
		return ""
	}
	if payloads.ResolveName(cfg[utils.Payload]) != "instance-cmd-check" {
		return ""
	}
	return shellTargetFromMetadata(cfg[utils.Metadata])
}

func shellTargetFromMetadata(metadata string) string {
	parts := argparse.SplitN(metadata, 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func rememberShellTarget(target, provider, source string) {
	target = strings.TrimSpace(target)
	if target == "" {
		return
	}
	hint := knownShellTargets[target]
	if hint.Provider == "" {
		hint.Provider = strings.TrimSpace(provider)
	}
	if hint.Source == "" {
		hint.Source = strings.TrimSpace(source)
	}
	knownShellTargets[target] = hint
}
