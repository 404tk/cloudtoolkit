package payloads

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type RDSAccountCheck struct{}

func (p RDSAccountCheck) Run(ctx context.Context, config map[string]string) {
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
	i, ok := loadInventory(config)
	if !ok {
		return
	}
	mgr, ok := i.Providers.(schema.DBManager)
	if !ok {
		logger.Error(fmt.Sprintf("%s does not support rds-account-check", i.Providers.Name()))
		return
	}
	mgr.DBManagement(action, args)
}

func (p RDSAccountCheck) Desc() string {
	return "Provision a read-only test database account in an authorized environment to validate database telemetry, investigation readiness, and control coverage."
}

func (p RDSAccountCheck) Sensitivity(metadata string) Sensitivity {
	data := argparse.Split(metadata)
	if len(data) < 2 {
		return Sensitivity{}
	}
	return Sensitivity{
		Level:      "destructive",
		ConfirmKey: "rds-account-check." + data[0],
		Resource:   data[1],
	}
}

func init() {
	registerPayload("rds-account-check", RDSAccountCheck{})
}
