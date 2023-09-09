package payloads

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type ExecuteCloudVMCommand struct{}

func (p ExecuteCloudVMCommand) Run(ctx context.Context, config map[string]string) {
	var instanceId, cmd string
	if metadata, ok := config["metadata"]; ok {
		data := strings.Split(metadata, " ")
		if len(data) < 2 {
			logger.Error("Execute `set metadata <instance-id> <cmd>`")
			return
		}
		instanceId = data[0]
		cmd = data[1]
	}
	i, err := inventory.New(config)
	if err != nil {
		logger.Error(err)
		return
	}
	i.Providers.ExecuteCloudVMCommand(instanceId, cmd)
}

func (p ExecuteCloudVMCommand) Desc() string {
	return "Run command on Cloud instance."
}

func init() {
	registerPayload("exec-command", ExecuteCloudVMCommand{})
}
