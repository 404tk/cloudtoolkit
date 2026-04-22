package console

import (
	"fmt"
	"sort"
	"strings"

	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
)

type HelpMode string

const (
	HelpModeRoot     HelpMode = "root"
	HelpModeProvider HelpMode = "provider"
	HelpModeShell    HelpMode = "shell"
)

type HelpContext struct {
	Mode       HelpMode
	Provider   string
	Payload    string
	InstanceID string
	DemoReplay bool
}

type helpTopic struct {
	Title    string
	Summary  string
	Usage    []string
	Details  []string
	Examples []string
}

type payloadHelp struct {
	MetadataSyntax   []string
	MetadataExamples []string
	SafetyNotes      []string
}

var helpTopicOrder = []string{
	"use",
	"sessions",
	"note",
	"show",
	"set",
	"run",
	"shell",
	"demo",
	"payload",
	"metadata",
}

var helpTopics = map[string]helpTopic{
	"use": {
		Title:   "Use",
		Summary: "Enter provider mode for an authorized cloud environment.",
		Usage: []string{
			"use <provider>",
		},
		Details: []string{
			"Provider mode enables `show`, `set`, `run`, and `shell` commands.",
			"Supported providers come from the built-in provider catalog.",
			"Use `sessions -i <id>` to reopen a cached provider session instead of starting from scratch.",
		},
		Examples: []string{
			"use aws",
			"use azure",
		},
	},
	"sessions": {
		Title:   "Sessions",
		Summary: "Inspect cached credentials, reopen provider mode, or remove cached entries.",
		Usage: []string{
			"sessions",
			"sessions -i <id>",
			"sessions -k <id>",
			"sessions -c [id]",
		},
		Details: []string{
			"`sessions` lists the cached credential entries known to the console.",
			"`-i` reopens provider mode with the selected cached session.",
			"`-k` removes a cached session entry, and `-c` validates one or all cached entries.",
		},
		Examples: []string{
			"sessions",
			"sessions -i 1",
			"sessions -c",
		},
	},
	"note": {
		Title:   "Note",
		Summary: "Attach a short label to a cached session entry.",
		Usage: []string{
			"note <session-id> <label>",
		},
		Details: []string{
			"Use this to mark sessions with environment or ownership context before reopening them later.",
			"The note is stored with the cached credential entry shown by `sessions`.",
		},
		Examples: []string{
			"note 1 lab-aws",
		},
	},
	"show": {
		Title:   "Show",
		Summary: "Display the current provider configuration or the visible validation payloads.",
		Usage: []string{
			"show options",
			"show payloads",
		},
		Details: []string{
			"`show options` prints the current provider settings, payload, and metadata fields.",
			"`show payloads` lists the visible validation payloads and their summaries.",
		},
		Examples: []string{
			"show options",
			"show payloads",
		},
	},
	"set": {
		Title:   "Set",
		Summary: "Update the current provider configuration, payload, or payload metadata.",
		Usage: []string{
			"set <option> <value>",
			"set payload <payload-name>",
			"set metadata <payload-specific-args>",
		},
		Details: []string{
			"`set payload <name>` resolves payload aliases to the canonical payload name.",
			"`set metadata` stores the payload-specific argument string used by `run`.",
			"Changing the payload may reset metadata defaults for payloads that provide them.",
		},
		Examples: []string{
			"set accesskey <value>",
			"set payload iam-user-check",
			"set metadata add demo-user 'TempPassw0rd!'",
		},
	},
	"run": {
		Title:   "Run",
		Summary: "Execute the selected validation payload with the current provider configuration.",
		Usage: []string{
			"run",
		},
		Details: []string{
			"`run` dispatches the active payload using the current provider settings and metadata.",
			"Sensitive payloads may prompt for confirmation before execution.",
			"Use `help payload <name>` before running a payload you have not used recently.",
		},
		Examples: []string{
			"run",
		},
	},
	"shell": {
		Title:   "Shell",
		Summary: "Open an authorized instance command validation shell for `instance-cmd-check`.",
		Usage: []string{
			"shell <instance-id>",
		},
		Details: []string{
			"Shell mode handles `help`, `clear`, `back`, `exit`, and `quit` locally.",
			"Any other input is treated as a remote instance command and dispatched through `instance-cmd-check`.",
			"Use shell mode only for owned, lab, or explicitly authorized instances.",
		},
		Examples: []string{
			"shell i-1234567890abcdef0",
		},
	},
	"demo": {
		Title:   "Demo",
		Summary: "Enable deterministic replay mode inside the current provider session.",
		Usage: []string{
			"demo",
		},
		Details: []string{
			"`demo` opens a nested provider mock session and changes the prompt to `ctk > <provider>[mock] >`.",
			"`exit`, `quit`, or `back` leave mock mode and return to the live provider session.",
			"After enabling demo replay, use the designated demo access key and secret key with the existing `set` commands.",
			"When the designated demo credentials match, supported payloads and shell sessions replay deterministic provider responses through the real provider and payload flow.",
			"If the credentials do not match, mock mode returns a natural authentication failure and does not silently fall through to live provider execution.",
		},
		Examples: []string{
			"use alibaba",
			"demo",
			"set accesskey LTAIxxxxxxxxxxxxEXAMPLE",
			"set secretkey EXAMPLExxxxxxxxxxxxxxxxKEY",
			"run",
			"exit",
		},
	},
	"payload": {
		Title:   "Payload",
		Summary: "Inspect visible validation payloads and get payload-specific guidance.",
		Usage: []string{
			"help payload",
			"help payload <payload-name>",
		},
		Details: []string{
			"`help payload` lists the visible payload catalog.",
			"`help payload <payload-name>` shows summary, metadata syntax, examples, and safety notes for one payload.",
		},
		Examples: []string{
			"help payload",
			"help payload cloudlist",
		},
	},
	"metadata": {
		Title:   "Metadata",
		Summary: "Inspect payload-specific metadata syntax and examples.",
		Usage: []string{
			"help metadata",
			"help metadata <payload-name>",
		},
		Details: []string{
			"`metadata` stores payload-specific arguments that are passed to the active payload.",
			"Use quotes when values include spaces or characters that should stay together as one argument.",
			"`help metadata <payload-name>` shows the exact syntax expected by that payload.",
		},
		Examples: []string{
			"help metadata",
			"help metadata iam-user-check",
		},
	},
}

var payloadHelpDocs = map[string]payloadHelp{
	"cloudlist": {
		MetadataSyntax: []string{
			"This payload does not require metadata.",
		},
		MetadataExamples: []string{
			"set payload cloudlist",
			"run",
		},
		SafetyNotes: []string{
			"Cloud asset inventory is read-oriented, but still use it only in owned, lab, or explicitly authorized environments.",
			"Provider credentials still need enough access to enumerate the resources you want to validate.",
		},
	},
	"iam-user-check": {
		MetadataSyntax: []string{
			"set metadata <action> <username> <password>",
			"`action` is typically `add` or `del`.",
		},
		MetadataExamples: []string{
			"set metadata add demo-user 'TempPassw0rd!'",
			"set metadata del demo-user cleanup-placeholder",
		},
		SafetyNotes: []string{
			"Use dedicated test identities and remove them after validation.",
			"Validate only in environments where creating or deleting IAM users is explicitly approved.",
		},
	},
	"bucket-check": {
		MetadataSyntax: []string{
			"set metadata <action> <bucket-name>",
			"`action` is typically `list` or `total`.",
		},
		MetadataExamples: []string{
			"set metadata list ctk-validation-bucket",
			"set metadata total ctk-validation-bucket",
		},
		SafetyNotes: []string{
			"Use buckets created for validation or otherwise explicitly approved for review.",
			"Reviewing bucket contents can expose sensitive data; align the test scope with the data owner first.",
		},
	},
	"event-check": {
		MetadataSyntax: []string{
			"set metadata <action> <scope>",
			"`action` is typically `dump`, and `<scope>` can be a source IP or `all`.",
		},
		MetadataExamples: []string{
			"set metadata dump all",
			"set metadata dump 198.51.100.24",
		},
		SafetyNotes: []string{
			"Use event review in environments where log access is approved.",
			"Treat event output as investigative data and handle it according to local retention and access policies.",
		},
	},
	"rds-account-check": {
		MetadataSyntax: []string{
			"set metadata <action> <instance-id>",
			"`action` is typically `useradd`.",
		},
		MetadataExamples: []string{
			"set metadata useradd rm-1234567890",
		},
		SafetyNotes: []string{
			"Run this only where creating validation database accounts is explicitly authorized.",
			"Remove temporary accounts after testing and confirm the expected database privilege scope before execution.",
		},
	},
	"instance-cmd-check": {
		MetadataSyntax: []string{
			"set metadata <instance-id> <cmd>",
			"`shell <instance-id>` wraps this payload and forwards all non-local input as `<cmd>`.",
		},
		MetadataExamples: []string{
			"set metadata i-1234567890abcdef0 whoami",
			"set metadata i-1234567890abcdef0 'id && hostname'",
			"shell i-1234567890abcdef0",
		},
		SafetyNotes: []string{
			"Use only on instances that are owned, lab-managed, or explicitly authorized for command validation.",
			"Remember that shell mode sends non-local input to the remote instance as a validation command.",
		},
	},
}

var helpOptionOrder = []string{
	utils.AccessKey,
	utils.SecretKey,
	utils.SecurityToken,
	utils.Region,
	utils.Version,
	utils.AzureClientId,
	utils.AzureClientSecret,
	utils.AzureTenantId,
	utils.AzureSubscriptionId,
	utils.GCPserviceAccountJSON,
}

var sensitiveHelpOptions = map[string]struct{}{
	utils.AccessKey:             {},
	utils.SecretKey:             {},
	utils.SecurityToken:         {},
	utils.AzureClientSecret:     {},
	utils.GCPserviceAccountJSON: {},
}

func currentHelpContext() HelpContext {
	ctx := HelpContext{
		Mode:       HelpModeRoot,
		Payload:    currentPayloadName(),
		DemoReplay: isDemoReplayActiveForCurrentProvider(),
	}
	if config != nil {
		ctx.Provider = config[utils.Provider]
	}
	if target := currentShellTarget(); target != "" {
		ctx.Mode = HelpModeShell
		ctx.InstanceID = target
		if ctx.Payload == "" {
			ctx.Payload = "instance-cmd-check"
		}
		return ctx
	}
	if ctx.Provider != "" {
		ctx.Mode = HelpModeProvider
		if ctx.Payload == "" {
			ctx.Payload = "cloudlist"
		}
	}
	return ctx
}

func currentShellTarget() string {
	if len(consoleStack) == 0 || instanceId == "" {
		return ""
	}
	return instanceId
}

func currentPayloadName() string {
	if config == nil {
		return ""
	}
	if name := config[utils.Payload]; name != "" {
		if _, resolved, ok := payloads.Lookup(name); ok {
			return resolved
		}
		return name
	}
	return ""
}

func renderContextHelp(ctx HelpContext) {
	switch ctx.Mode {
	case HelpModeProvider:
		renderProviderHelp(ctx)
	case HelpModeShell:
		renderShellHelp(ctx)
	default:
		renderRootHelp()
	}
}

func renderRootHelp() {
	var b strings.Builder
	writeHeader(&b, "CloudToolKit Help")
	writeLines(&b, "Global commands:", []string{
		"help [topic]            Show context-aware help.",
		"use <provider>          Enter provider mode.",
		"sessions                List cached sessions.",
		"note <id> <label>       Add a short note to a cached session.",
		"clear                   Clear the screen.",
		"exit                    Exit the console.",
	})
	writeLines(&b, "Provider mode commands:", []string{
		"show options            Review provider configuration.",
		"show payloads           Review visible validation payloads.",
		"set <option> <value>    Update provider options or payload metadata.",
		"demo                    Enable deterministic replay for supported providers.",
		"run                     Execute the active payload.",
		"shell <instance-id>     Open an instance command validation shell.",
	})
	writeLines(&b, "Onboarding:", []string{
		"1. use <provider>",
		"2. set the required provider options",
		"3. show payloads",
		"4. set payload <payload-name>",
		"5. help payload <payload-name>",
		"6. run",
	})
	writeLines(&b, "More help:", []string{
		"help use",
		"help sessions",
		"help demo",
		"help payload",
		"help metadata",
	})
	writeLines(&b, "Responsible use:", []string{
		"Use CloudToolKit only in owned, lab, or explicitly authorized environments.",
	})
	fmt.Print(b.String())
}

func renderProviderHelp(ctx HelpContext) {
	var b strings.Builder
	writeHeader(&b, "Provider Help")
	writeLines(&b, "Current context:", []string{
		fmt.Sprintf("mode: %s", providerHelpModeLabel(ctx)),
		fmt.Sprintf("provider: %s", helpValueOrDefault(ctx.Provider, "(not set)")),
		fmt.Sprintf("payload: %s", helpValueOrDefault(ctx.Payload, "cloudlist")),
		fmt.Sprintf("demo replay: %s", demoReplayStatus(ctx.DemoReplay)),
	})
	writeLines(&b, "Current config summary:", providerConfigSummaryLines())

	required := providerRequiredOptionLines()
	if len(required) == 0 {
		required = []string{"No provider is currently selected."}
	}
	writeLines(&b, "Required options:", required)
	writeLines(&b, "Available commands:", []string{
		commandListSummary(HelpModeProvider, ctx.DemoReplay),
	})
	writeLines(&b, "Next recommended commands:", providerRecommendedCommands(ctx))
	writeLines(&b, "Responsible use:", []string{
		"Validate only in owned, lab, or explicitly authorized environments.",
	})
	fmt.Print(b.String())
}

func renderShellHelp(ctx HelpContext) {
	var b strings.Builder
	writeHeader(&b, "Shell Help")
	writeLines(&b, "Current context:", []string{
		"mode: shell",
		fmt.Sprintf("provider: %s", helpValueOrDefault(ctx.Provider, "(not set)")),
		fmt.Sprintf("target instance: %s", helpValueOrDefault(ctx.InstanceID, "(not set)")),
		fmt.Sprintf("payload: %s", helpValueOrDefault(ctx.Payload, "instance-cmd-check")),
		fmt.Sprintf("demo replay: %s", demoReplayStatus(ctx.DemoReplay)),
	})
	writeLines(&b, "Local shell commands:", []string{
		"help [topic]            Show local help without sending a remote command.",
		"clear                   Clear the screen.",
		"exit                    Close shell mode and return to provider mode.",
	})
	if ctx.DemoReplay {
		writeLines(&b, "Replay behavior:", []string{
			"Any other input is replayed locally from deterministic demo data for the current target.",
			"No real remote command execution will occur while demo replay is active and the designated demo credentials are configured.",
		})
	} else {
		writeLines(&b, "Remote command behavior:", []string{
			"Any other input is treated as a remote instance command for the current authorized target.",
			"Use `help shell` or `help payload instance-cmd-check` if you need payload details before sending a command.",
		})
	}
	if ctx.DemoReplay {
		writeLines(&b, "Responsible use:", []string{
			"Use demo replay only for walkthroughs, demos, and validation dry-runs in owned, lab, or explicitly authorized environments.",
		})
	} else {
		writeLines(&b, "Responsible use:", []string{
			"Run remote commands only against owned, lab, or explicitly authorized instances.",
		})
	}
	fmt.Print(b.String())
}

func renderTopicHelp(_ HelpContext, topic helpTopic) {
	var b strings.Builder
	writeHeader(&b, topic.Title+" Help")
	writeLines(&b, "Summary:", []string{topic.Summary})
	writeLines(&b, "Usage:", topic.Usage)
	writeLines(&b, "Details:", topic.Details)
	writeLines(&b, "Examples:", topic.Examples)
	fmt.Print(b.String())
}

func renderPayloadCatalogHelp(ctx HelpContext) {
	var b strings.Builder
	writeHeader(&b, "Payload Help")
	if ctx.Provider != "" {
		writeLines(&b, "Current provider:", []string{ctx.Provider})
	}
	if ctx.Payload != "" {
		writeLines(&b, "Current payload:", []string{ctx.Payload})
	}
	lines := make([]string, 0, len(payloads.Visible()))
	for _, entry := range payloads.Visible() {
		lines = append(lines, fmt.Sprintf("%s - %s", entry.Name, entry.Payload.Desc()))
	}
	writeLines(&b, "Visible payloads:", lines)
	writeLines(&b, "Inspect one payload:", []string{
		"help payload <payload-name>",
		"help metadata <payload-name>",
	})
	fmt.Print(b.String())
}

func renderMetadataOverviewHelp(ctx HelpContext) {
	var b strings.Builder
	writeHeader(&b, "Metadata Help")
	writeLines(&b, "Summary:", []string{
		"`metadata` stores payload-specific arguments that are passed to the active payload.",
	})
	writeLines(&b, "Syntax rules:", []string{
		"set metadata <payload-specific-args>",
		"Whitespace separates arguments.",
		"Use single or double quotes when values must contain spaces.",
		"Use `help metadata <payload-name>` for payload-specific syntax and examples.",
	})
	if ctx.Payload != "" {
		writeLines(&b, "Current payload:", []string{ctx.Payload})
	}
	fmt.Print(b.String())
}

func renderPayloadHelp(ctx HelpContext, name string, metadataOnly bool) {
	entry, resolved, ok := findVisiblePayload(name)
	if !ok {
		fmt.Printf("No payload help available for %q.\n", name)
		fmt.Println("Use `help payload` to review the visible payload catalog.")
		return
	}

	doc := payloadHelpDocs[resolved]
	var b strings.Builder
	if metadataOnly {
		writeHeader(&b, "Metadata Help: "+resolved)
	} else {
		writeHeader(&b, "Payload Help: "+resolved)
	}
	writeLines(&b, "Summary:", []string{entry.Payload.Desc()})
	if ctx.Provider != "" {
		writeLines(&b, "Current provider:", []string{ctx.Provider})
	}
	writeLines(&b, "Metadata syntax:", doc.MetadataSyntax)
	writeLines(&b, "Metadata examples:", doc.MetadataExamples)
	writeLines(&b, "Safety notes:", append([]string{
		"Use CloudToolKit only in owned, lab, or explicitly authorized environments.",
	}, doc.SafetyNotes...))
	fmt.Print(b.String())
}

func findVisiblePayload(name string) (payloads.Entry, string, bool) {
	resolved := payloads.ResolveName(name)
	for _, entry := range payloads.Visible() {
		if entry.Name == resolved {
			return entry, resolved, true
		}
	}
	return payloads.Entry{}, "", false
}

func providerConfigSummaryLines() []string {
	if config == nil {
		return []string{"No provider configuration is loaded."}
	}

	lines := make([]string, 0, len(config)+2)
	for _, key := range providerConfigKeys() {
		lines = append(lines, fmt.Sprintf("%s: %s", key, summarizeConfigValue(key, config[key])))
	}
	lines = append(lines,
		fmt.Sprintf("%s: %s", utils.Metadata, summarizeConfigValue(utils.Metadata, config[utils.Metadata])),
	)
	return lines
}

func providerRequiredOptionLines() []string {
	if config == nil {
		return nil
	}
	lines := make([]string, 0)
	for _, key := range providerConfigKeys() {
		if isOptionalHelpOption(key) {
			continue
		}
		status := "missing"
		if strings.TrimSpace(config[key]) != "" {
			status = "set"
		}
		desc := optionsDesc[key]
		lines = append(lines, fmt.Sprintf("%s: %s - %s", key, status, desc))
	}
	if len(lines) == 0 {
		return []string{"All required provider options are currently set."}
	}
	return lines
}

func providerRecommendedCommands(ctx HelpContext) []string {
	canExit := commandAvailable(HelpModeProvider, ctx.DemoReplay, "exit")
	canDemo := commandAvailable(HelpModeProvider, ctx.DemoReplay, "demo")

	if config == nil {
		return []string{
			"use <provider>",
			"sessions -i <id>",
		}
	}

	missing := make([]string, 0)
	for _, key := range providerConfigKeys() {
		if isOptionalHelpOption(key) {
			continue
		}
		if strings.TrimSpace(config[key]) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		commands := []string{"show options"}
		if canExit {
			commands = append(commands, "exit")
		}
		for _, key := range missing {
			commands = append(commands, fmt.Sprintf("set %s <value>", key))
		}
		if canDemo {
			commands = append(commands, "demo")
		}
		commands = append(commands,
			"show payloads",
			"help payload",
		)
		return commands
	}

	commands := []string{
		"show payloads",
		"set payload <payload-name>",
		"help payload <payload-name>",
	}
	if canExit {
		commands = append([]string{"exit"}, commands...)
	}
	if canDemo {
		commands = append(commands, "demo")
	}
	if ctx.Payload == "instance-cmd-check" {
		commands = append(commands,
			"help metadata instance-cmd-check",
			"shell <instance-id>",
		)
	}
	commands = append(commands, "run")
	return commands
}

func providerConfigKeys() []string {
	if config == nil {
		return nil
	}

	keys := make([]string, 0, len(config))
	seen := make(map[string]struct{})
	for _, key := range helpOptionOrder {
		if _, ok := config[key]; ok {
			keys = append(keys, key)
			seen[key] = struct{}{}
		}
	}

	extra := make([]string, 0)
	for key := range config {
		switch key {
		case utils.Provider, utils.Payload, utils.Metadata:
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		extra = append(extra, key)
	}
	sort.Strings(extra)
	return append(keys, extra...)
}

func isOptionalHelpOption(key string) bool {
	desc := optionsDesc[key]
	return strings.Contains(desc, "Optional") || strings.Contains(desc, "Default:")
}

func summarizeConfigValue(key, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "(empty)"
	}
	if _, ok := sensitiveHelpOptions[key]; ok {
		return "(set)"
	}
	if len(value) > 72 {
		return value[:69] + "..."
	}
	return value
}

func helpValueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func demoReplayStatus(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func commandListSummary(mode HelpMode, demoReplay bool) string {
	return strings.Join(commandNamesForContext(mode, demoReplay), ", ")
}

func providerHelpModeLabel(ctx HelpContext) string {
	if ctx.DemoReplay {
		return "provider[mock]"
	}
	return "provider"
}

func writeHeader(b *strings.Builder, title string) {
	fmt.Fprintf(b, "%s\n", title)
	fmt.Fprintf(b, "%s\n\n", strings.Repeat("=", len(title)))
}

func writeLines(b *strings.Builder, title string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(b, "%s\n", title)
	for _, line := range lines {
		fmt.Fprintf(b, "  %s\n", line)
	}
	b.WriteString("\n")
}
