package huawei

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/utils"
)

func init() {
	registry.Register("huawei", registry.Spec{
		Options: []registry.Option{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []registry.Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "cn-north-4", Description: "Beijing 4"},
			{Text: "cn-east-3", Description: "Shanghai 1"},
			{Text: "cn-south-1", Description: "Guangzhou"},
			{Text: "ap-southeast-1", Description: "Hong Kong"},
			{Text: "eu-west-101", Description: "Dublin"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket", "iam-role", "bucket-acl", "event"},
	})
}
