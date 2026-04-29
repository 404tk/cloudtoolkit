// Package registry is the single source of truth for provider self-description.
// Each pkg/providers/<x>/ registers its Spec at init() time; runner/catalog and
// other consumers read aggregated views from here.
//
// Dependency direction is intentional and load-bearing:
//   - registry imports only standard library + utils
//   - pkg/providers/<x>/ imports registry (one-way, via init())
//   - runner/catalog and runner/{console,headless} import registry
//
// Anything that needs to know "which provider" must live above registry; the
// package itself is provider-agnostic.
package registry

import (
	"sort"
	"strings"
	"sync"

	"github.com/404tk/cloudtoolkit/utils"
)

// Suggestion is a single autocomplete entry (text + helper description) used
// for region pickers and similar UI surfaces.
type Suggestion struct {
	Text        string
	Description string
}

// Option describes one provider configuration field (e.g. accesskey, region).
type Option struct {
	Name        string
	Description string
	Default     string
	Required    bool
	Sensitive   bool
}

// Spec is the full self-description of a provider: its config options,
// region suggestions, and the capability identifiers it claims to support.
// Capability strings line up with runner/catalog payload capabilities
// ("cloudlist" / "iam" / "bucket" / "event" / "vm" / "database").
type Spec struct {
	Options      []Option
	Regions      []Suggestion
	Capabilities []string
}

var (
	mu    sync.RWMutex
	specs = map[string]Spec{}
)

// Register stores spec for the named provider. Re-registering the same name
// overwrites; intended to be called from init() so the order across providers
// does not matter.
func Register(name string, spec Spec) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	specs[name] = spec
}

// Lookup returns the spec for name if registered.
func Lookup(name string) (Spec, bool) {
	mu.RLock()
	defer mu.RUnlock()
	spec, ok := specs[strings.TrimSpace(name)]
	return spec, ok
}

// Names returns all registered provider names sorted alphabetically.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultConfig builds a config map of option name -> default value.
func DefaultConfig(name string) (map[string]string, bool) {
	spec, ok := Lookup(name)
	if !ok {
		return nil, false
	}
	cfg := make(map[string]string, len(spec.Options))
	for _, option := range spec.Options {
		cfg[option.Name] = option.Default
	}
	return cfg, true
}

// Capabilities returns a copy of the provider's declared capability list.
func Capabilities(name string) []string {
	spec, ok := Lookup(name)
	if !ok {
		return nil
	}
	return append([]string(nil), spec.Capabilities...)
}

// SupportsCapability reports whether the named provider declares the given capability.
func SupportsCapability(provider, capability string) bool {
	for _, item := range Capabilities(provider) {
		if item == capability {
			return true
		}
	}
	return false
}

// Regions returns a copy of the provider's region suggestions.
func Regions(name string) []Suggestion {
	spec, ok := Lookup(name)
	if !ok {
		return nil
	}
	out := make([]Suggestion, len(spec.Regions))
	copy(out, spec.Regions)
	return out
}

// Options returns a copy of the provider's option list.
func Options(name string) []Option {
	spec, ok := Lookup(name)
	if !ok {
		return nil
	}
	out := make([]Option, len(spec.Options))
	copy(out, spec.Options)
	return out
}

// OptionDescription returns the rendered description (with default / optional
// hints) for an option name aggregated across all registered providers, plus
// the two payload-level pseudo-options (payload, metadata). Returns "" when
// the option is unknown.
func OptionDescription(name string) string {
	return optionDescriptions()[name]
}

// SensitiveOption reports whether any registered provider marks this option
// name as sensitive (e.g. a secret that should not be logged or echoed).
func SensitiveOption(name string) bool {
	_, ok := sensitiveOptions()[name]
	return ok
}

// OptionNames returns every option name known to the registry sorted
// alphabetically, including payload-level pseudo-options.
func OptionNames() []string {
	descs := optionDescriptions()
	names := make([]string, 0, len(descs))
	for name := range descs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func optionDescriptions() map[string]string {
	mu.RLock()
	defer mu.RUnlock()
	items := make(map[string]string)
	for _, spec := range specs {
		for _, option := range spec.Options {
			desc := option.Description
			switch {
			case option.Default != "":
				desc += " (Default: " + option.Default + ")"
			case !option.Required:
				desc += " (Optional)"
			}
			items[option.Name] = desc
		}
	}
	items[utils.Payload] = "Validation payload (Default: cloudlist)"
	items[utils.Metadata] = "Set the payload with additional arguments (Optional)"
	return items
}

func sensitiveOptions() map[string]struct{} {
	mu.RLock()
	defer mu.RUnlock()
	items := make(map[string]struct{})
	for _, spec := range specs {
		for _, option := range spec.Options {
			if option.Sensitive {
				items[option.Name] = struct{}{}
			}
		}
	}
	return items
}
