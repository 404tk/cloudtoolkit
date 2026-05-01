package aws

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/utils"
)

func init() {
	registry.Register("aws", registry.Spec{
		Options: []registry.Option{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []registry.Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "us-east-1", Description: "N. Virginia"},
			{Text: "us-east-2", Description: "Ohio"},
			{Text: "us-west-1", Description: "N. California"},
			{Text: "us-west-2", Description: "Oregon"},
			{Text: "ap-east-1", Description: "Hong Kong"},
			{Text: "ap-southeast-1", Description: "Singapore"},
			{Text: "ap-southeast-2", Description: "Sydney"},
			{Text: "ap-northeast-1", Description: "Tokyo"},
			{Text: "ap-northeast-2", Description: "Seoul"},
			{Text: "eu-west-1", Description: "Ireland"},
			{Text: "eu-central-1", Description: "Frankfurt"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket", "iam-role", "bucket-acl", "vm", "event"},
	})
}
