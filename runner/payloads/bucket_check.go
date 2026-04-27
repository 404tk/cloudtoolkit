package payloads

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/argparse"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type BucketCheck struct{}

func (p BucketCheck) Run(ctx context.Context, config map[string]string) {
	var action, bucketname string
	if metadata, ok := config["metadata"]; ok {
		data := argparse.Split(metadata)
		if len(data) < 2 {
			logger.Error("Execute `set metadata dump <bucket>`")
			return
		}
		action = data[0]
		bucketname = data[1]
	}
	i, ok := loadInventory(config)
	if !ok {
		return
	}
	mgr, ok := i.Providers.(schema.BucketManager)
	if !ok {
		logger.Error(fmt.Sprintf("%s does not support bucket-check", i.Providers.Name()))
		return
	}
	mgr.BucketDump(ctx, action, bucketname)
	logger.Info("Done.")
}

func (p BucketCheck) Desc() string {
	return "Review bucket contents in an authorized test environment to validate storage visibility and investigation workflows."
}

func init() {
	registerPayload("bucket-check", BucketCheck{})
}
