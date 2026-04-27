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

type ResultProducer interface {
	Result(context.Context, map[string]string) (any, error)
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
