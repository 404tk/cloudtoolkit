package payloads

import (
	"context"
	"fmt"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
)

type CloudList struct{}

func (p CloudList) Run(ctx context.Context, config map[string]string) {
	i, err := inventory.New(config)
	if err != nil {
		logger.Error(err)
		return
	}

	resources, err := i.Providers.Resources(ctx)
	if err != nil {
		logger.Error(err)
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
					utils.WriteLog(path, tag+" results:")
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
		pprint(len(resources.Logs), "Log Service", resources.Logs)
		if resources.Sms.DailySize > 0 {
			msg := fmt.Sprintf("The total number of SMS messages sent today is %v.", resources.Sms.DailySize)
			logger.Info(msg)
		}
		if utils.DoSave {
			logger.Info(fmt.Sprintf("Output written to [%s]", path))
		} else {
			logger.Info("Done.")
		}
	}
}

func (p CloudList) Desc() string {
	return "Getting Assets from Cloud Providers to augment Attack Surface Management efforts."
}

func init() {
	registerPayload("cloudlist", CloudList{})
}
