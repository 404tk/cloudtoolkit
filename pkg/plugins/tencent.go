//go:build !no_tencent

package plugins

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Tencent struct{}

func (p Tencent) Check(block schema.Options) (schema.Provider, error) {
	return tencent.New(block)
}

func (p Tencent) Desc() string {
	return "Tencent Cloud"
}

func init() {
	registerProvider("tencent", Tencent{})
}
