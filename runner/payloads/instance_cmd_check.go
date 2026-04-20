package payloads

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/audit"
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
	i, err := inventory.New(config)
	if err != nil {
		logger.Error(err)
		return
	}
	execer, ok := i.Providers.(schema.VMExecutor)
	if !ok {
		logger.Error(fmt.Sprintf("%s does not support instance-cmd-check", i.Providers.Name()))
		return
	}
	record := audit.Record{
		Provider:  config[utils.Provider],
		Operation: "instance-cmd-check",
		Target:    instanceId,
	}
	if decoded, err := base64.StdEncoding.DecodeString(cmd); err == nil {
		record.Args = string(decoded)
	} else {
		record.Args = cmd
	}
	audit.Log(record)
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
