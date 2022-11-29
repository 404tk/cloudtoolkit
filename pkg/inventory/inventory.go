package inventory

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/404tk/cloudtoolkit/pkg/plugins"
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
		} else if IsNil(provider) {
			return inventory, errors.New("It maybe Huawei Cloud SDK panic.")
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
func nameToProvider(name string, block schema.OptionBlock) (schema.Provider, error) {
	fmt.Println(plugins.Providers)
	if v, ok := plugins.Providers[name]; ok {
		return v.Check(block)
	}
	return nil, fmt.Errorf("invalid provider name found: %s", name)
}
