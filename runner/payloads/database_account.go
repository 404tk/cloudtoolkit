package payloads

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type DatabaseAccount struct{}

func (p DatabaseAccount) Run(ctx context.Context, config map[string]string) {
	var action, args string
	if metadata, ok := config["metadata"]; ok {
		data := strings.Split(metadata, " ")
		if len(data) < 2 {
			logger.Error("Execute `set metadata useradd <instance-id>`")
			return
		}
		action = data[0]
		args = data[1]
	}
	i, err := inventory.New(config)
	if err != nil {
		logger.Error(err)
		return
	}
	i.Providers.DBManagement(action, args)
}

func (p DatabaseAccount) Desc() string {
	return "Add an account with read-only permission for the Cloud database instance."
}

func init() {
	registerPayload("database-account", DatabaseAccount{})
}
