package alibaba

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/utils"
)

func init() {
	registry.Register("alibaba", registry.Spec{
		Options: []registry.Option{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []registry.Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "cn-beijing", Description: "Beijing"},
			{Text: "cn-hangzhou", Description: "Hangzhou"},
			{Text: "cn-shanghai", Description: "Shanghai"},
			{Text: "cn-shenzhen", Description: "Shenzhen"},
			{Text: "cn-hongkong", Description: "Hong Kong"},
			{Text: "ap-southeast-1", Description: "Singapore"},
			{Text: "us-east-1", Description: "Virginia"},
			{Text: "eu-central-1", Description: "Frankfurt"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket", "event", "vm", "database"},
	})
}
