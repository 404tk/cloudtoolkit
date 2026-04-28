package payloads

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type EventCheck struct{}

func (p EventCheck) Run(ctx context.Context, config map[string]string) {
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
	i, ok := loadInventory(config)
	if !ok {
		return
	}
	reader, ok := i.Providers.(schema.EventReader)
	if !ok {
		logger.Error(fmt.Sprintf("%s does not support event-check", i.Providers.Name()))
		return
	}
	reader.EventDump(action, sourceIp)
}

func (p EventCheck) Desc() string {
	return "Review cloud security events from an authorized environment to validate alert context and investigation workflows."
}

func (p EventCheck) Sensitivity(metadata string) Sensitivity {
	data := argparse.Split(metadata)
	if len(data) < 1 || data[0] != "whitelist" {
		return Sensitivity{}
	}
	resource := ""
	if len(data) >= 2 {
		resource = data[1]
	}
	return Sensitivity{
		Level:      "mutate",
		ConfirmKey: "event-check.whitelist",
		Resource:   resource,
	}
}

func init() {
	registerPayload("event-check", EventCheck{})
}
