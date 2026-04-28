package payloads

import (
	"context"
	"errors"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/logger"
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
				fmt.Printf("\n%-70s\t%-10s\n", "Key", "Size")
				fmt.Printf("%-70s\t%-10s\n", "---", "----")
				for _, obj := range result.Objects {
					label := obj.Key
					if result.BucketName == "" && obj.BucketName != "" {
						label = obj.BucketName + "/" + obj.Key
					}
					fmt.Printf("%-70s\t%-10s\n", label, utils.ParseBytes(obj.Size))
				}
				fmt.Println()
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
