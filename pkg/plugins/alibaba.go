//go:build !no_alibaba

package plugins

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Alibaba struct{}

func (p Alibaba) Check(block schema.Options) (schema.Provider, error) {
	return alibaba.New(block)
}

func (p Alibaba) Desc() string {
	return "Alibaba Cloud"
}

func init() {
	registerProvider("alibaba", Alibaba{})
}
