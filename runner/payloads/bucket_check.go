package payloads

import (
	"context"
	"errors"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/argparse"
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
	RunStructured(ctx, config, p)
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
		err := fmt.Errorf("%s does not support bucket-check", i.Providers.Name())
		return nil, NewResultError(nil, CodeUnsupported, err)
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
		return results, NewResultError(results, CodeExecutionFailed, opErr)
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
