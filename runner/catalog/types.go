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

type PayloadSpec struct {
	Capability  string
	Sensitivity string
}
