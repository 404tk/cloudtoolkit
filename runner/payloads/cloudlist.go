package payloads

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
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
	RunStructured(ctx, config, p)
}

func (p CloudList) Result(ctx context.Context, config map[string]string) (any, error) {
	result, _, err := p.result(ctx, config)
	if err != nil {
		return nil, err
	}
	if len(result.Errors) > 0 {
		return result, NewResultError(result, CodePartialFailure, errors.New("one or more cloud resources could not be enumerated"))
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
		err := fmt.Errorf("%s does not support cloud asset inventory", i.Providers.Name())
		return nil, cloudListExecution{}, NewResultError(nil, CodeUnsupported, err)
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
