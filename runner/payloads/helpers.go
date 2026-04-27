package payloads

import (
	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func inventoryFromConfig(config map[string]string) (*inventory.Inventory, error) {
	return inventory.New(config)
}

func loadInventory(config map[string]string) (*inventory.Inventory, bool) {
	i, err := inventoryFromConfig(config)
	if err != nil {
		logger.Error(err)
		return nil, false
	}
	return i, true
}
