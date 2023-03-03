package inventory

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/404tk/cloudtoolkit/pkg/plugins"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Inventory is an inventory of providers
type Inventory struct {
	Providers schema.Provider
}

// New creates a new inventory of providers
func New(options schema.Options) (*Inventory, error) {
	inventory := &Inventory{}

	value, ok := options.GetMetadata(utils.Provider)
	if !ok {
		return inventory, nil
	}
	provider, err := nameToProvider(value, options)
	if err != nil {
		return inventory, err
	} else if IsNil(provider) {
		return inventory, errors.New("It maybe Huawei Cloud SDK panic.")
	}
	inventory.Providers = provider
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
func nameToProvider(name string, block schema.Options) (schema.Provider, error) {
	if v, ok := plugins.Providers[name]; ok {
		return v.Check(block)
	}
	return nil, fmt.Errorf("invalid provider name found: %s", name)
}
