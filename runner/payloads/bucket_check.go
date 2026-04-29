package payloads

import (
	"context"
	"errors"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
)

type BucketCheck struct{}

type BucketCheckResult struct {
	Provider    string                `json:"provider"`
	Action      string                `json:"action"`
	BucketName  string                `json:"bucket_name,omitempty"`
	ObjectCount int64                 `json:"object_count,omitempty"`
	Objects     []schema.BucketObject `json:"objects,omitempty"`
	Message     string                `json:"message,omitempty"`
	Status      string                `json:"status"`
	Error       string                `json:"error,omitempty"`
}

type bucketAction struct {
	Action     string
	BucketName string
}

func (p BucketCheck) Run(ctx context.Context, config map[string]string) {
	resultAny, err := p.Result(ctx, config)
	if err != nil && resultAny == nil {
		logger.Error(err.Error())
		return
	}

	results, ok := resultAny.([]BucketCheckResult)
	if !ok {
		logger.Error("Invalid result type")
		return
	}
	for _, result := range results {
		if result.Status == "error" {
			if result.Error != "" {
				logger.Error(result.Error)
			}
			continue
		}

		switch result.Action {
		case "list":
			if result.BucketName != "" {
				logger.Warning(fmt.Sprintf("%d objects found in %s.", result.ObjectCount, result.BucketName))
			} else if result.Message != "" {
				logger.Warning(result.Message)
			}
			if len(result.Objects) > 0 {
				type objectRow struct {
					Key  string `table:"Key"`
					Size string `table:"Size"`
				}
				rows := make([]objectRow, 0, len(result.Objects))
				for _, obj := range result.Objects {
					label := obj.Key
					if result.BucketName == "" && obj.BucketName != "" {
						label = obj.BucketName + "/" + obj.Key
					}
					rows = append(rows, objectRow{
						Key:  label,
						Size: utils.ParseBytes(obj.Size),
					})
				}
				table.Output(rows)
			}
		case "total":
			if result.BucketName != "" {
				logger.Warning(fmt.Sprintf("%s has %d objects.", result.BucketName, result.ObjectCount))
			} else if result.Message != "" {
				logger.Warning(result.Message)
			}
		default:
			if result.Message != "" {
				logger.Warning(result.Message)
			}
		}
	}
}

func (p BucketCheck) Result(ctx context.Context, config map[string]string) (any, error) {
	parsed, err := parseBucketAction(config["metadata"])
	if err != nil {
		return nil, err
	}
	i, err := inventoryFromConfig(config)
	if err != nil {
		return nil, err
	}
	mgr, ok := i.Providers.(schema.BucketManager)
	if !ok {
		return nil, fmt.Errorf("%s does not support bucket-check", i.Providers.Name())
	}

	bucketResults, opErr := mgr.BucketDump(ctx, parsed.Action, parsed.BucketName)
	results := make([]BucketCheckResult, 0, len(bucketResults))
	for _, bucketResult := range bucketResults {
		results = append(results, BucketCheckResult{
			Provider:    i.Providers.Name(),
			Action:      bucketResult.Action,
			BucketName:  bucketResult.BucketName,
			ObjectCount: bucketResult.ObjectCount,
			Objects:     bucketResult.Objects,
			Message:     bucketResult.Message,
			Status:      "success",
		})
	}
	if opErr != nil {
		if len(results) == 0 {
			results = append(results, BucketCheckResult{
				Provider:   i.Providers.Name(),
				Action:     parsed.Action,
				BucketName: parsed.BucketName,
				Status:     "error",
				Error:      opErr.Error(),
			})
		} else {
			results = append(results, BucketCheckResult{
				Provider:   i.Providers.Name(),
				Action:     parsed.Action,
				BucketName: parsed.BucketName,
				Status:     "error",
				Error:      opErr.Error(),
			})
		}
		return results, NewResultError(results, 4, opErr)
	}
	return results, nil
}

func (p BucketCheck) Desc() string {
	return "Review bucket contents in an authorized test environment to validate storage visibility and investigation workflows."
}

func (p BucketCheck) Capability() string {
	return "bucket"
}

func (p BucketCheck) Help() HelpDoc {
	return HelpDoc{
		MetadataSyntax: []string{
			"set metadata <action> <bucket-name>",
			"`action` is typically `list` or `total`.",
		},
		MetadataExamples: []string{
			"set metadata list ctk-validation-bucket",
			"set metadata total ctk-validation-bucket",
		},
		MetadataSuggestions: []Suggestion{
			{Text: "list <bucket-name>", Description: "review object listings inside one authorized bucket"},
			{Text: "total <bucket-name>", Description: "count objects in one bucket"},
		},
		SafetyNotes: []string{
			"Use buckets created for validation or otherwise explicitly approved for review.",
			"Reviewing bucket contents can expose sensitive data; align the test scope with the data owner first.",
		},
	}
}

func parseBucketAction(metadata string) (bucketAction, error) {
	data := argparse.Split(metadata)
	if len(data) < 2 {
		return bucketAction{}, errors.New("invalid metadata format: expected 'list <bucket>' or 'total <bucket>'")
	}
	return bucketAction{
		Action:     data[0],
		BucketName: data[1],
	}, nil
}

func init() {
	registerPayload("bucket-check", BucketCheck{})
}
