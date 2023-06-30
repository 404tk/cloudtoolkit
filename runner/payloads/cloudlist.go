package payloads

import (
	"context"
	"fmt"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/utils/table"
)

type CloudList struct{}

func (p CloudList) Run(ctx context.Context, config map[string]string) {
	i, err := inventory.New(config)
	if err != nil {
		log.Println(err)
		return
	}

	resources, err := i.Providers.Resources(ctx)
	if err != nil {
		log.Println("[Failed]", err.Error())
		// return
	}
	select {
	case <-ctx.Done():
		return
	default:
		pprint := func(len int, tag string, res interface{}) {
			if len > 0 {
				fmt.Println(tag, "results:")
				table.Output(res)
			}
		}

		pprint(len(resources.Hosts), "Hosts", resources.Hosts)
		pprint(len(resources.Storages), "Storages", resources.Storages)
		pprint(len(resources.Users), "Users", resources.Users)
		pprint(len(resources.Databases), "Databases", resources.Databases)
		pprint(len(resources.Sms.Signs), "SMS Signs", resources.Sms.Signs)
		pprint(len(resources.Sms.Templates), "SMS Templates", resources.Sms.Templates)
		if resources.Sms.DailySize > 0 {
			fmt.Printf("[*] The total number of SMS messages sent today is %v.\n", resources.Sms.DailySize)
		}

		log.Println("[+] Done.")
	}
}

func (p CloudList) Desc() string {
	return "Getting Assets from Cloud Providers to augment Attack Surface Management efforts."
}

func init() {
	registerPayload("cloudlist", CloudList{})
}
