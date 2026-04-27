package catalog

import "strings"

var payloadSpecs = map[string]PayloadSpec{
	"cloudlist": {
		Capability:  "cloudlist",
		Sensitivity: "read",
		Help: PayloadHelp{
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
	},
	"iam-user-check": {
		Capability:  "iam",
		Sensitivity: "destructive",
		MetadataTemplates: []Suggestion{
			{Text: "add <username> <password>", Description: "create a validation IAM user"},
			{Text: "del <username>", Description: "remove a validation IAM user"},
		},
		Help: PayloadHelp{
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
	},
	"bucket-check": {
		Capability:  "bucket",
		Sensitivity: "read",
		MetadataTemplates: []Suggestion{
			{Text: "list <bucket-name>", Description: "review bucket contents in an authorized environment"},
			{Text: "total <bucket-name>", Description: "count objects in a bucket"},
		},
		Help: PayloadHelp{
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
	},
	"event-check": {
		Capability:  "event",
		Sensitivity: "mixed",
		MetadataTemplates: []Suggestion{
			{Text: "dump all", Description: "review all relevant events"},
			{Text: "dump <source-ip>", Description: "review events for one source IP"},
		},
		Help: PayloadHelp{
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
	},
	"rds-account-check": {
		Capability:  "database",
		Sensitivity: "destructive",
		MetadataTemplates: []Suggestion{
			{Text: "useradd <instance-id>", Description: "provision a validation database account"},
		},
		Help: PayloadHelp{
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
	},
	"instance-cmd-check": {
		Capability:  "vm",
		Sensitivity: "destructive",
		MetadataTemplates: []Suggestion{
			{Text: "<instance-id> <cmd>", Description: "direct metadata syntax for one remote command; prefer `shell <instance-id>` for interactive use"},
		},
		Help: PayloadHelp{
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
	},
}

func PayloadSpecFor(name string) (PayloadSpec, bool) {
	spec, ok := payloadSpecs[strings.TrimSpace(name)]
	return spec, ok
}

func PayloadCapability(name string) string {
	spec, ok := PayloadSpecFor(name)
	if !ok {
		return ""
	}
	return spec.Capability
}

func PayloadSensitivity(name string) string {
	spec, ok := PayloadSpecFor(name)
	if !ok {
		return ""
	}
	return spec.Sensitivity
}

func PayloadMetadataSuggestions(name string) []Suggestion {
	spec, ok := PayloadSpecFor(name)
	if !ok {
		return nil
	}
	out := make([]Suggestion, len(spec.MetadataTemplates))
	copy(out, spec.MetadataTemplates)
	return out
}

func PayloadHelpFor(name string) (PayloadHelp, bool) {
	spec, ok := PayloadSpecFor(name)
	if !ok {
		return PayloadHelp{}, false
	}
	return spec.Help, true
}
