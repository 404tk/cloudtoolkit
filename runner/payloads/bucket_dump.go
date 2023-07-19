package payloads

import (
	"context"
	"log"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
)

type BucketDump struct{}

func (p BucketDump) Run(ctx context.Context, config map[string]string) {
	i, err := inventory.New(config)
	if err != nil {
		log.Println(err)
		return
	}

	var action, bucketname string
	if metadata, ok := config["metadata"]; ok {
		data := strings.Split(metadata, " ")
		if len(data) >= 2 {
			action = data[0]
			bucketname = data[1]
		}
	}
	i.Providers.BucketDump(ctx, action, bucketname)
	log.Println("[+] Done.")
}

func (p BucketDump) Desc() string {
	return "Quickly enumerate buckets to look for loot."
}

func init() {
	registerPayload("bucket-dump", BucketDump{})
}
