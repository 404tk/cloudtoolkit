package payloads

import (
	"context"
	"sort"
	"strings"

	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Payload interface {
	Run(context.Context, map[string]string)
	Desc() string
}

type Suggestion struct {
	Text        string
	Description string
}

type HelpDoc struct {
	MetadataSyntax      []string
	MetadataExamples    []string
	MetadataSuggestions []Suggestion
	SafetyNotes         []string
}

type HelpProvider interface {
	Help() HelpDoc
}

type ResultProducer interface {
	Result(context.Context, map[string]string) (any, error)
}

// CapabilityProvider lets a payload declare which provider capability it needs
// (e.g. "iam", "bucket", "vm"). headless uses this to short-circuit before
// calling into the provider for a payload it cannot satisfy. Payloads that do
// not implement this interface are treated as universally applicable.
type CapabilityProvider interface {
	Capability() string
}

type ResultError interface {
	error
	ResultPayload() any
	ExitCode() int
}

type structuredResultError struct {
	payload  any
	err      error
	exitCode int
}

func (e structuredResultError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e structuredResultError) ResultPayload() any {
	return e.payload
}

func (e structuredResultError) ExitCode() int {
	if e.exitCode <= 0 {
		return 1
	}
	return e.exitCode
}

func NewResultError(payload any, exitCode int, err error) error {
	if err == nil {
		return nil
	}
	return structuredResultError{
		payload:  payload,
		err:      err,
		exitCode: exitCode,
	}
}

type Entry struct {
	Name    string
	Payload Payload
}

var Payloads = make(map[string]Payload)

func registerPayload(pName string, p Payload) {
	if _, ok := Payloads[pName]; ok {
		logger.Error("Payload registered multiple times:", pName)
	}
	Payloads[pName] = p
}

func Lookup(name string) (Payload, string, bool) {
	name = strings.TrimSpace(name)
	p, ok := Payloads[name]
	return p, name, ok
}

func Visible() []Entry {
	names := make([]string, 0, len(Payloads))
	for name := range Payloads {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]Entry, 0, len(names))
	for _, name := range names {
		entries = append(entries, Entry{Name: name, Payload: Payloads[name]})
	}
	return entries
}

func HelpFor(name string) (HelpDoc, bool) {
	p, _, ok := Lookup(name)
	if !ok {
		return HelpDoc{}, false
	}
	helpProvider, ok := p.(HelpProvider)
	if !ok {
		return HelpDoc{}, false
	}
	return cloneHelpDoc(helpProvider.Help()), true
}

// PayloadCapability returns the provider capability this payload requires, or
// "" if the payload does not implement CapabilityProvider (treated as
// universally applicable). Mirrors the DescribeSensitivity pattern.
func PayloadCapability(name string) string {
	p, _, ok := Lookup(name)
	if !ok {
		return ""
	}
	cp, ok := p.(CapabilityProvider)
	if !ok {
		return ""
	}
	return cp.Capability()
}

func MetadataSuggestions(name string) []Suggestion {
	doc, ok := HelpFor(name)
	if !ok {
		return nil
	}
	return append([]Suggestion(nil), doc.MetadataSuggestions...)
}

func cloneHelpDoc(doc HelpDoc) HelpDoc {
	doc.MetadataSyntax = append([]string(nil), doc.MetadataSyntax...)
	doc.MetadataExamples = append([]string(nil), doc.MetadataExamples...)
	doc.MetadataSuggestions = append([]Suggestion(nil), doc.MetadataSuggestions...)
	doc.SafetyNotes = append([]string(nil), doc.SafetyNotes...)
	return doc
}
