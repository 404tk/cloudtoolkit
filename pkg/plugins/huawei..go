package plugins

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Huawei struct{}

func (p Huawei) Check(block schema.OptionBlock) (schema.Provider, error) {
	return huawei.New(block)
}

func (p Huawei) Desc() string {
	return "Huawei Cloud"
}

func init() {
	registerProvider("huawei", Huawei{})
}
