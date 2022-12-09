package payloads

import (
	"log"
)

type Payload interface {
	Run(map[string]string)
	Desc() string
}

var Payloads = make(map[string]Payload)

func registerPayload(pName string, p Payload) {
	if _, ok := Payloads[pName]; ok {
		log.Println("Payloads multiple registration:", pName)
	}
	Payloads[pName] = p
}
