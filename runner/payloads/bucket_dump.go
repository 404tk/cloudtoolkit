package payloads

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type BucketDump struct{}

func (p BucketDump) Run(ctx context.Context, config map[string]string) {
	var action, bucketname string
	if metadata, ok := config["metadata"]; ok {
		data := strings.Split(metadata, " ")
		if len(data) < 2 {
			logger.Error("Execute `set metadata dump <bucket>`")
			return
		}
		action = data[0]
		bucketname = data[1]
	}
	i, err := inventory.New(config)
	if err != nil {
		logger.Error(err)
		return
	}
	i.Providers.BucketDump(ctx, action, bucketname)
	logger.Info("Done.")
}

func (p BucketDump) Desc() string {
	return "Quickly enumerate buckets to look for loot."
}

func init() {
	registerPayload("bucket-dump", BucketDump{})
}
