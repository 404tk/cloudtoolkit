package providers

import (
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba"
	alireplay "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/replay"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine"
	volcreplay "github.com/404tk/cloudtoolkit/pkg/providers/volcengine/replay"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Info struct {
	Name string
	Desc string
}

type entry struct {
	info Info
	new  func(schema.Options) (schema.Provider, error)
}

var catalog = []entry{
	{
		info: Info{Name: "alibaba", Desc: "Alibaba Cloud"},
		new: func(block schema.Options) (schema.Provider, error) {
			if demoreplay.IsActiveForProvider("alibaba") {
				return alibaba.NewWithConfig(block, alireplay.ClientConfig())
			}
			return alibaba.New(block)
		},
	},
	{
		info: Info{Name: "aws", Desc: "Amazon Web Service"},
		new: func(block schema.Options) (schema.Provider, error) {
			return aws.New(block)
		},
	},
	{
		info: Info{Name: "tencent", Desc: "Tencent Cloud"},
		new: func(block schema.Options) (schema.Provider, error) {
			return tencent.New(block)
		},
	},
	{
		info: Info{Name: "huawei", Desc: "Huawei Cloud"},
		new: func(block schema.Options) (schema.Provider, error) {
			return huawei.New(block)
		},
	},
	{
		info: Info{Name: "azure", Desc: "Microsoft Azure"},
		new: func(block schema.Options) (schema.Provider, error) {
			return azure.New(block)
		},
	},
	{
		info: Info{Name: "volcengine", Desc: "Volcengine"},
		new: func(block schema.Options) (schema.Provider, error) {
			if demoreplay.IsActiveForProvider("volcengine") {
				return volcengine.NewWithConfig(block, volcreplay.ClientConfig())
			}
			return volcengine.New(block)
		},
	},
	{
		info: Info{Name: "jdcloud", Desc: "JDCloud"},
		new: func(block schema.Options) (schema.Provider, error) {
			return jdcloud.New(block)
		},
	},
	{
		info: Info{Name: "gcp", Desc: "Google Cloud Platform"},
		new: func(block schema.Options) (schema.Provider, error) {
			return gcp.New(block)
		},
	},
}

var catalogByName = func() map[string]entry {
	items := make(map[string]entry, len(catalog))
	for _, item := range catalog {
		items[item.info.Name] = item
	}
	return items
}()

func Supported() []Info {
	items := make([]Info, 0, len(catalog))
	for _, item := range catalog {
		items = append(items, item.info)
	}
	return items
}

func Supports(name string) bool {
	_, ok := catalogByName[strings.TrimSpace(name)]
	return ok
}

func New(name string, block schema.Options) (schema.Provider, error) {
	item, ok := catalogByName[strings.TrimSpace(name)]
	if !ok {
		return nil, fmt.Errorf("invalid provider name found: %s", name)
	}
	return item.new(block)
}
