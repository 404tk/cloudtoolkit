package payloads

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
)

type BucketACLCheck struct{}

type BucketACLCheckResult struct {
	Provider   string                  `json:"provider"`
	Action     string                  `json:"action"`
	Container  string                  `json:"container,omitempty"`
	Level      string                  `json:"level,omitempty"`
	Containers []bucketACLEntryJSON    `json:"containers,omitempty"`
	Message    string                  `json:"message,omitempty"`
	Status     string                  `json:"status"`
	Error      string                  `json:"error,omitempty"`
}

type bucketACLEntryJSON struct {
	Account   string `json:"account,omitempty"`
	Container string `json:"container"`
	Level     string `json:"level"`
}

type bucketACLAction struct {
	Action    string
	Container string
	Level     string
}

func (p BucketACLCheck) Run(ctx context.Context, config map[string]string) {
	resultAny, err := p.Result(ctx, config)
	if err != nil && resultAny == nil {
		logger.Error(err.Error())
		return
	}
	result, ok := resultAny.(BucketACLCheckResult)
	if !ok {
		logger.Error("Invalid result type")
		return
	}
	if result.Status == "error" {
		logger.Error(result.Error)
		return
	}

	if len(result.Containers) > 0 {
		type aclRow struct {
			Account   string `table:"Account"`
			Container string `table:"Container"`
			Level     string `table:"Public Access"`
		}
		rows := make([]aclRow, 0, len(result.Containers))
		for _, entry := range result.Containers {
			rows = append(rows, aclRow{
				Account:   entry.Account,
				Container: entry.Container,
				Level:     entry.Level,
			})
		}
		table.Output(rows)
	}
	if result.Message != "" {
		logger.Warning(result.Message)
	}
}

func (p BucketACLCheck) Result(ctx context.Context, config map[string]string) (any, error) {
	parsed, err := parseBucketACLAction(config["metadata"])
	if err != nil {
		return nil, err
	}

	i, err := inventoryFromConfig(config)
	if err != nil {
		return nil, err
	}

	mgr, ok := i.Providers.(schema.BucketACLManager)
	if !ok {
		return nil, fmt.Errorf("%s does not support bucket-acl-check", i.Providers.Name())
	}

	aclResult, err := mgr.BucketACL(ctx, parsed.Action, parsed.Container, parsed.Level)

	result := BucketACLCheckResult{
		Provider:  i.Providers.Name(),
		Action:    parsed.Action,
		Container: parsed.Container,
		Level:     parsed.Level,
	}
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, NewResultError(result, 4, err)
	}

	if aclResult.Container != "" {
		result.Container = aclResult.Container
	}
	if aclResult.Level != "" {
		result.Level = aclResult.Level
	}
	result.Message = aclResult.Message
	for _, entry := range aclResult.Containers {
		result.Containers = append(result.Containers, bucketACLEntryJSON{
			Account:   entry.Account,
			Container: entry.Container,
			Level:     entry.Level,
		})
	}
	result.Status = "success"
	return result, nil
}

func (p BucketACLCheck) Desc() string {
	return "Toggle storage container public-access settings in an authorized environment to validate detection coverage for unintended data exposure."
}

func (p BucketACLCheck) Capability() string {
	return "bucket-acl"
}

func (p BucketACLCheck) Help() HelpDoc {
	return HelpDoc{
		MetadataSyntax: []string{
			"set metadata <action> [container] [level]",
			"`action` is typically `audit`, `expose`, or `unexpose`.",
			"`level` is only meaningful for `expose`: provider-specific (e.g. Azure `Blob` / `Container`).",
		},
		MetadataExamples: []string{
			"set metadata audit",
			"set metadata audit ctk-demo-container",
			"set metadata expose ctk-demo-container Blob",
			"set metadata unexpose ctk-demo-container",
		},
		MetadataSuggestions: []Suggestion{
			{Text: "audit", Description: "list public-access state for every container in scope"},
			{Text: "audit <container>", Description: "show public-access state for a single container"},
			{Text: "expose <container> <level>", Description: "set public-access on a container to validate exposure detection"},
			{Text: "unexpose <container>", Description: "revert public-access back to private"},
		},
		SafetyNotes: []string{
			"Public access on storage containers can leak data; only run against containers explicitly created for validation.",
			"Always run `unexpose` (or revert via cloud console) immediately after detection coverage is confirmed.",
		},
	}
}

func (p BucketACLCheck) Sensitivity(metadata string) Sensitivity {
	data := argparse.Split(metadata)
	if len(data) == 0 {
		return Sensitivity{}
	}
	switch data[0] {
	case "expose", "unexpose":
	default:
		return Sensitivity{}
	}
	resource := ""
	if len(data) >= 2 {
		resource = data[1]
	}
	return Sensitivity{
		Level:      "destructive",
		ConfirmKey: "bucket-acl-check." + data[0],
		Resource:   resource,
	}
}

func parseBucketACLAction(metadata string) (bucketACLAction, error) {
	data := argparse.Split(metadata)
	if len(data) == 0 {
		return bucketACLAction{}, errors.New("invalid metadata format: expected 'audit [container]', 'expose <container> [level]', or 'unexpose <container>'")
	}
	action := bucketACLAction{Action: strings.ToLower(data[0])}
	switch action.Action {
	case "audit":
		if len(data) >= 2 {
			action.Container = data[1]
		}
	case "expose":
		if len(data) < 2 {
			return bucketACLAction{}, errors.New("invalid metadata format: expected 'expose <container> [level]'")
		}
		action.Container = data[1]
		if len(data) >= 3 {
			action.Level = data[2]
		}
	case "unexpose":
		if len(data) < 2 {
			return bucketACLAction{}, errors.New("invalid metadata format: expected 'unexpose <container>'")
		}
		action.Container = data[1]
	default:
		return bucketACLAction{}, fmt.Errorf("unsupported action %q: expected audit / expose / unexpose", data[0])
	}
	return action, nil
}

func init() {
	registerPayload("bucket-acl-check", BucketACLCheck{})
}
