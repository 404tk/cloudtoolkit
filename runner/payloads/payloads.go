package payloads

import (
	"context"
	"sort"

	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Payload interface {
	Run(context.Context, map[string]string)
	Desc() string
}

type Entry struct {
	Name    string
	Payload Payload
}

var Payloads = make(map[string]Payload)
var aliases = make(map[string]string)

func registerPayload(pName string, p Payload) {
	if _, ok := Payloads[pName]; ok {
		logger.Error("Payload registered multiple times:", pName)
	}
	Payloads[pName] = p
}

func registerAlias(alias, canonical string) {
	if _, ok := aliases[alias]; ok {
		logger.Error("Payload alias registered multiple times:", alias)
	}
	aliases[alias] = canonical
}

func ResolveName(name string) string {
	if canonical, ok := aliases[name]; ok {
		return canonical
	}
	return name
}

func Lookup(name string) (Payload, string, bool) {
	resolved := ResolveName(name)
	p, ok := Payloads[resolved]
	return p, resolved, ok
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
