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

type EventDump struct{}

func (p EventDump) Run(ctx context.Context, config map[string]string) {
	var action, sourceIp string
	if metadata, ok := config["metadata"]; ok {
		data := argparse.Split(metadata)
		if len(data) < 2 {
			logger.Error("Execute `set metadata dump all`")
			return
		}
		action = data[0]
		sourceIp = data[1]
	}
	i, err := inventory.New(config)
	if err != nil {
		logger.Error(err)
		return
	}
	reader, ok := i.Providers.(schema.EventReader)
	if !ok {
		logger.Error(fmt.Sprintf("%s does not support event-dump", i.Providers.Name()))
		return
	}
	audit.Log(audit.Record{
		Provider:  config[utils.Provider],
		Operation: "event-dump." + action,
		Target:    sourceIp,
	})
	reader.EventDump(action, sourceIp)
	logger.Info("Done.")
}

func (p EventDump) Desc() string {
	return "Obtain alarm events of the cloud platform."
}

func init() {
	registerPayload("event-dump", EventDump{})
}
