//go:build !no_jdcloud

package plugins

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type JDCloud struct{}

func (p JDCloud) Check(block schema.Options) (schema.Provider, error) {
	return jdcloud.New(block)
}

func (p JDCloud) Desc() string {
	return "JDCloud"
}

func init() {
	registerProvider("jdcloud", JDCloud{})
}
