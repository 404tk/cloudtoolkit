package plugins

import (
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Provider interface {
	Check(block schema.Options) (schema.Provider, error)
	Desc() string
}

var Providers = make(map[string]Provider)

func registerProvider(pName string, p Provider) {
	if _, ok := Providers[pName]; ok {
		log.Println("Provider multiple registration:", pName)
	}
	Providers[pName] = p
}
