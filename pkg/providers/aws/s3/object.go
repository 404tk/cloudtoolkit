package s3

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/s3"
)

func (d *S3Provider) ListObjects(buckets map[string]string) {
	for b, r := range buckets {
		d.Session.Config.Region = &r
		client := s3.New(d.Session)
		var limit = int64(100) // Do not display more yet.
		input := &s3.ListObjectsV2Input{Bucket: &b, MaxKeys: &limit}
		resp, err := client.ListObjectsV2(input)
		if err != nil {
			log.Printf("[-] List Objects in %s failed: %s\n", b, err.Error())
			continue
		}

		if len(resp.Contents) == 0 {
			log.Printf("[-] No Objects found in %s.\n", b)
			continue
		}
		log.Printf("[+] %d objects found in %s.\n", len(resp.Contents), b)

		fmt.Printf("\n%-70s\t%-10s\n", "Key", "Size")
		fmt.Printf("%-70s\t%-10s\n", "---", "----")
		for _, object := range resp.Contents {
			fmt.Printf("%-70s\t%-10s\n",
				*object.Key, fmt.Sprintf("%v bytes", *object.Size))
		}
		fmt.Println()
	}
}
