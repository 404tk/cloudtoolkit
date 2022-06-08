package inventory

import (
	"fmt"
	"reflect"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// Inventory is an inventory of providers
type Inventory struct {
	Providers []schema.Provider
}

// New creates a new inventory of providers
func New(options schema.Options) (*Inventory, error) {
	inventory := &Inventory{}

	for _, block := range options {
		value, ok := block.GetMetadata("provider")
		if !ok {
			return inventory, nil
		}
		provider, err := nameToProvider(value, block)
		if err != nil {
			return inventory, err
		}
		inventory.Providers = append(inventory.Providers, provider)
	}
	return inventory, nil
}

func IsNil(i schema.Provider) bool {
	vi := reflect.ValueOf(i)
	if vi.Kind() == reflect.Ptr {
		return vi.IsNil()
	}
	return false
}

// nameToProvider returns the provider for a name
func nameToProvider(value string, block schema.OptionBlock) (schema.Provider, error) {
	switch value {
	case "aws":
		return aws.New(block)
	case "azure":
		return azure.New(block)
	case "alibaba":
		return alibaba.New(block)
	case "tencent":
		return tencent.New(block)
	case "huawei":
		return huawei.New(block)
	default:
		return nil, fmt.Errorf("invalid provider name found: %s", value)
	}
}
