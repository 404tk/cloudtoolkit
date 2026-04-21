package payloads

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type InstanceCmdCheck struct{}

func (p InstanceCmdCheck) Run(ctx context.Context, config map[string]string) {
	var instanceId, cmd string
	if metadata, ok := config["metadata"]; ok {
		data := argparse.SplitN(metadata, 2)
		if len(data) < 2 {
			logger.Error("Execute `set metadata <instance-id> <cmd>`")
			return
		}
		instanceId = data[0]
		cmd = data[1]
	}
	i, ok := loadInventory(config)
	if !ok {
		return
	}
	execer, ok := i.Providers.(schema.VMExecutor)
	if !ok {
		logger.Error(fmt.Sprintf("%s does not support instance-cmd-check", i.Providers.Name()))
		return
	}
	execer.ExecuteCloudVMCommand(instanceId, cmd)
}

func (p InstanceCmdCheck) Desc() string {
	return "Run an authorized validation command on a cloud instance to generate telemetry for detection and investigation verification."
}

func init() {
	registerPayload("instance-cmd-check", InstanceCmdCheck{})
	registerAlias("instance-command", "instance-cmd-check")
	registerAlias("exec-command", "instance-cmd-check")
}
