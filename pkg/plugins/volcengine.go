//go:build !no_volcengine

package plugins

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Volcengine struct{}

func (p Volcengine) Check(block schema.Options) (schema.Provider, error) {
	return volcengine.New(block)
}

func (p Volcengine) Desc() string {
	return "Volcengine"
}

func init() {
	registerProvider("volcengine", Volcengine{})
}
