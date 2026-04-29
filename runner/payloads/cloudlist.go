package payloads

import (
	"context"
	"fmt"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
)

type CloudList struct{}

type CloudListResult struct {
	Provider    string                 `json:"provider"`
	Hosts       []schema.Host          `json:"hosts,omitempty"`
	Storages    []schema.Storage       `json:"storages,omitempty"`
	Users       []schema.User          `json:"users,omitempty"`
	Databases   []schema.Database      `json:"databases,omitempty"`
	Domains     []schema.Domain        `json:"domains,omitempty"`
	Logs        []schema.Log           `json:"logs,omitempty"`
	SMS         schema.Sms             `json:"sms,omitempty"`
	Errors      []schema.ResourceError `json:"errors,omitempty"`
	OutputFiles []string               `json:"output_files,omitempty"`
}

type cloudListExecution struct {
	provider  string
	path      string
	resources schema.Resources
}

func (p CloudList) Run(ctx context.Context, config map[string]string) {
	result, exec, err := p.result(ctx, config)
	if err != nil {
		logger.Error(err)
		return
	}
	if result == nil {
		return
	}
	select {
	case <-ctx.Done():
		return
	default:
		e := env.From(ctx)
		printGroup := func(tag string, items interface{}) {
			logger.Warning(tag + " results:")
			table.Output(items)
			if e.LogEnable {
				utils.WriteLog(exec.path, tag+" results:")
				table.FileOutput(exec.path, items)
			}
		}

		if len(result.Hosts) > 0 {
			printGroup("Hosts", result.Hosts)
		}
		if len(result.Storages) > 0 {
			printGroup("Storages", result.Storages)
		}
		if len(result.Users) > 0 {
			printGroup("Users", result.Users)
		}
		if len(result.Databases) > 0 {
			printGroup("Databases", result.Databases)
		}
		for _, domain := range result.Domains {
			if len(domain.Records) == 0 {
				continue
			}
			printGroup("Domain "+domain.DomainName, domain.Records)
		}
		if len(result.Logs) > 0 {
			printGroup("Log Service", result.Logs)
		}

		if len(result.SMS.Signs) > 0 {
			printGroup("SMS Signs", result.SMS.Signs)
		}
		if len(result.SMS.Templates) > 0 {
			printGroup("SMS Templates", result.SMS.Templates)
		}
		if result.SMS.DailySize > 0 {
			msg := fmt.Sprintf("The total number of SMS messages sent today is %v.", result.SMS.DailySize)
			logger.Info(msg)
		}

		for _, item := range result.Errors {
			logger.Error(fmt.Sprintf("%s failed: %s", item.Scope, item.Message))
		}
		if e.LogEnable {
			logger.Info(fmt.Sprintf("Output written to [%s]", exec.path))
		}
	}
}

func (p CloudList) Result(ctx context.Context, config map[string]string) (any, error) {
	result, _, err := p.result(ctx, config)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p CloudList) result(ctx context.Context, config map[string]string) (*CloudListResult, cloudListExecution, error) {
	i, err := inventoryFromConfig(config)
	if err != nil {
		return nil, cloudListExecution{}, err
	}
	enum, ok := i.Providers.(schema.Enumerator)
	if !ok {
		return nil, cloudListExecution{}, fmt.Errorf("%s does not support cloud asset inventory", i.Providers.Name())
	}

	resources, err := enum.Resources(ctx)
	if err != nil && len(resources.Errors) == 0 {
		return nil, cloudListExecution{}, err
	}
	exec := cloudListExecution{
		provider:  i.Providers.Name(),
		resources: resources,
	}
	if e := env.From(ctx); e.LogEnable {
		filename := time.Now().Format("20060102150405.log")
		exec.path = fmt.Sprintf("%s/%s_cloudlist_%s", e.LogDir, i.Providers.Name(), filename)
	}
	result := buildCloudListResult(exec)
	return &result, exec, nil
}

func buildCloudListResult(exec cloudListExecution) CloudListResult {
	result := CloudListResult{
		Provider: exec.provider,
		SMS:      exec.resources.Sms,
		Errors:   append([]schema.ResourceError(nil), exec.resources.Errors...),
	}
	if exec.path != "" {
		result.OutputFiles = []string{exec.path}
	}
	for _, asset := range exec.resources.Assets {
		switch v := asset.(type) {
		case schema.Host:
			result.Hosts = append(result.Hosts, v)
		case schema.Storage:
			result.Storages = append(result.Storages, v)
		case schema.User:
			result.Users = append(result.Users, v)
		case schema.Database:
			result.Databases = append(result.Databases, v)
		case schema.Domain:
			result.Domains = append(result.Domains, v)
		case schema.Log:
			result.Logs = append(result.Logs, v)
		}
	}
	return result
}

func (p CloudList) Desc() string {
	return "Enumerate cloud assets in authorized environments to verify CSPM and CNAPP inventory coverage, telemetry quality, and investigation readiness."
}

func (p CloudList) Capability() string {
	return "cloudlist"
}

func (p CloudList) Help() HelpDoc {
	return HelpDoc{
		MetadataSyntax: []string{
			"This payload does not require metadata.",
		},
		MetadataExamples: []string{
			"set payload cloudlist",
			"run",
		},
		SafetyNotes: []string{
			"Cloud asset inventory is read-oriented, but still use it only in owned, lab, or explicitly authorized environments.",
			"Provider credentials still need enough access to enumerate the resources you want to validate.",
		},
	}
}

func init() {
	registerPayload("cloudlist", CloudList{})
}
