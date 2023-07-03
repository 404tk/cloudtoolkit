package payloads

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/utils"
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
		filename := time.Now().Format("20060102150405.log")
		path := fmt.Sprintf("%s/%s_cloudlist_%s", utils.LogDir, i.Providers.Name(), filename)
		pprint := func(len int, tag string, res interface{}) {
			if len > 0 {
				fmt.Println(tag, "results:")
				table.Output(res)
				if utils.DoSave {
					table.FileOutput(path, res)
				}
			}
		}

		pprint(len(resources.Hosts), "Hosts", resources.Hosts)
		for _, domain := range resources.Domains {
			pprint(len(domain.Records), "Domain "+domain.DomainName, domain.Records)
		}
		pprint(len(resources.Storages), "Storages", resources.Storages)
		pprint(len(resources.Users), "Users", resources.Users)
		pprint(len(resources.Databases), "Databases", resources.Databases)
		pprint(len(resources.Sms.Signs), "SMS Signs", resources.Sms.Signs)
		pprint(len(resources.Sms.Templates), "SMS Templates", resources.Sms.Templates)
		if resources.Sms.DailySize > 0 {
			fmt.Printf("[*] The total number of SMS messages sent today is %v.\n", resources.Sms.DailySize)
		}
		if utils.DoSave {
			log.Printf("[+] Output written to [%s]\n", path)
		} else {
			log.Println("[+] Done.")
		}
	}
}

func (p CloudList) Desc() string {
	return "Getting Assets from Cloud Providers to augment Attack Surface Management efforts."
}

func init() {
	registerPayload("cloudlist", CloudList{})
}
