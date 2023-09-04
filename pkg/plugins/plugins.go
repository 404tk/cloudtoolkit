package plugins

import (
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Provider interface {
	Check(block schema.Options) (schema.Provider, error)
	Desc() string
}

var Providers = make(map[string]Provider)

func registerProvider(pName string, p Provider) {
	if _, ok := Providers[pName]; ok {
		logger.Error("Provider multiple registration:", pName)
	}
	Providers[pName] = p
}
