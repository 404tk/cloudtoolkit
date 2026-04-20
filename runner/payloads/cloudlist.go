package payloads

import (
	"context"
	"fmt"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
)

type CloudList struct{}

// assetPrintOrder keeps asset-inventory output stable regardless of the map
// iteration order returned by Resources.Grouped().
var assetPrintOrder = []struct {
	key, label string
}{
	{schema.AssetHost, "Hosts"},
	{schema.AssetStorage, "Storages"},
	{schema.AssetUser, "Users"},
	{schema.AssetDatabase, "Databases"},
	{schema.AssetDomain, "Domains"},
	{schema.AssetLog, "Log Service"},
}

func (p CloudList) Run(ctx context.Context, config map[string]string) {
	i, err := inventory.New(config)
	if err != nil {
		logger.Error(err)
		return
	}
	enum, ok := i.Providers.(schema.Enumerator)
	if !ok {
		logger.Error(fmt.Sprintf("%s does not support cloud asset inventory", i.Providers.Name()))
		return
	}

	resources, err := enum.Resources(ctx)
	if err != nil && len(resources.Errors) == 0 {
		logger.Error(err)
		return
	}
	select {
	case <-ctx.Done():
		return
	default:
		filename := time.Now().Format("20060102150405.log")
		path := fmt.Sprintf("%s/%s_cloudlist_%s", utils.LogDir, i.Providers.Name(), filename)
		printGroup := func(tag string, items interface{}) {
			fmt.Println(tag, "results:")
			table.Output(items)
			if utils.DoSave {
				utils.WriteLog(path, tag+" results:")
				table.FileOutput(path, items)
			}
		}

		groups := resources.Grouped()
		for _, entry := range assetPrintOrder {
			items := groups[entry.key]
			if len(items) == 0 {
				continue
			}
			if entry.key == schema.AssetDomain {
				for _, a := range items {
					domain, ok := a.(schema.Domain)
					if !ok || len(domain.Records) == 0 {
						continue
					}
					printGroup("Domain "+domain.DomainName, domain.Records)
				}
				continue
			}
			printGroup(entry.label, items)
		}

		if len(resources.Sms.Signs) > 0 {
			printGroup("SMS Signs", resources.Sms.Signs)
		}
		if len(resources.Sms.Templates) > 0 {
			printGroup("SMS Templates", resources.Sms.Templates)
		}
		if resources.Sms.DailySize > 0 {
			msg := fmt.Sprintf("The total number of SMS messages sent today is %v.", resources.Sms.DailySize)
			logger.Info(msg)
		}

		for _, item := range resources.Errors {
			logger.Error(fmt.Sprintf("%s failed: %s", item.Scope, item.Message))
		}
		if utils.DoSave {
			logger.Info(fmt.Sprintf("Output written to [%s]", path))
			if len(resources.Errors) > 0 {
				logger.Error("Cloud asset enumeration completed with partial errors.")
			}
		} else if len(resources.Errors) > 0 {
			logger.Error("Cloud asset enumeration completed with partial errors.")
		} else {
			logger.Info("Done.")
		}
	}
}

func (p CloudList) Desc() string {
	return "Enumerate cloud assets in authorized environments to verify CSPM and CNAPP inventory coverage, telemetry quality, and investigation readiness."
}

func init() {
	registerPayload("cloudlist", CloudList{})
}
