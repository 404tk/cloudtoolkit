package payloads

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/audit"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type DatabaseAccount struct{}

func (p DatabaseAccount) Run(ctx context.Context, config map[string]string) {
	var action, args string
	if metadata, ok := config["metadata"]; ok {
		data := argparse.Split(metadata)
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
	mgr, ok := i.Providers.(schema.DBManager)
	if !ok {
		logger.Error(fmt.Sprintf("%s does not support database-account", i.Providers.Name()))
		return
	}
	audit.Log(audit.Record{
		Provider:  config[utils.Provider],
		Operation: "database-account." + action,
		Target:    args,
	})
	mgr.DBManagement(action, args)
}

func (p DatabaseAccount) Desc() string {
	return "Add an account with read-only permission for the Cloud database instance."
}

func init() {
	registerPayload("database-account", DatabaseAccount{})
}
