package catalog

type Suggestion struct {
	Text        string
	Description string
}

type ProviderOption struct {
	Name        string
	Description string
	Default     string
	Required    bool
	Sensitive   bool
}

type ProviderSpec struct {
	Options      []ProviderOption
	Regions      []Suggestion
	Capabilities []string
}

type PayloadHelp struct {
	MetadataSyntax   []string
	MetadataExamples []string
	SafetyNotes      []string
}

type PayloadSpec struct {
	Capability        string
	Sensitivity       string
	MetadataTemplates []Suggestion
	Help              PayloadHelp
}
