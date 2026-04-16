package payloads

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/audit"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type ExecuteCloudVMCommand struct{}

func (p ExecuteCloudVMCommand) Run(ctx context.Context, config map[string]string) {
	var instanceId, cmd string
	if metadata, ok := config["metadata"]; ok {
		data := strings.SplitN(metadata, " ", 2)
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
		logger.Error(fmt.Sprintf("%s does not support exec-command", i.Providers.Name()))
		return
	}
	record := audit.Record{
		Provider:  config[utils.Provider],
		Operation: "exec-command",
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

func (p ExecuteCloudVMCommand) Desc() string {
	return "Run command on Cloud instance."
}

func init() {
	registerPayload("exec-command", ExecuteCloudVMCommand{})
}
