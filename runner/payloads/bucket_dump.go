package payloads

import (
	"log"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/inventory"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type BucketDump struct{}

func (p BucketDump) Run(config map[string]string) {
	inventory, err := inventory.New(schema.Options{config})
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

	for _, provider := range inventory.Providers {
		provider.BucketDump(action, bucketname)
	}
	log.Println("[+] Done.")
}

func (p BucketDump) Desc() string {
	return "Quickly enumerate buckets to look for loot."
}

func init() {
	registerPayload("bucket-dump", BucketDump{})
}
