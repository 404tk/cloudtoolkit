//go:build !no_azure

package plugins

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/azure"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Azure struct{}

func (p Azure) Check(block schema.Options) (schema.Provider, error) {
	return azure.New(block)
}

func (p Azure) Desc() string {
	return "Microsoft Azure"
}

func init() {
	registerProvider("azure", Azure{})
}
