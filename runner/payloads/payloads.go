package payloads

import (
	"context"

	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Payload interface {
	Run(context.Context, map[string]string)
	Desc() string
}

var Payloads = make(map[string]Payload)

func registerPayload(pName string, p Payload) {
	if _, ok := Payloads[pName]; ok {
		logger.Error("Payloads multiple registration:", pName)
	}
	Payloads[pName] = p
}
