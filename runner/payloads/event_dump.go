package payloads

import (
	"context"
	"log"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
)

type EventDump struct{}

func (p EventDump) Run(ctx context.Context, config map[string]string) {
	i, err := inventory.New(config)
	if err != nil {
		log.Println(err)
		return
	}

	var action, sourceIp string
	if metadata, ok := config["metadata"]; ok {
		data := strings.Split(metadata, " ")
		if len(data) >= 2 {
			action = data[0]
			sourceIp = data[1]
		}
	}
	i.Providers.EventDump(action, sourceIp)
	log.Println("[+] Done.")
}

func (p EventDump) Desc() string {
	return "Obtain alarm events of the cloud platform."
}

func init() {
	registerPayload("event-dump", EventDump{})
}
