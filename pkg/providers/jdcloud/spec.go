package jdcloud

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/utils"
)

func init() {
	registry.Register("jdcloud", registry.Spec{
		Options: []registry.Option{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []registry.Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "cn-north-1", Description: "Beijing"},
			{Text: "cn-east-2", Description: "Shanghai"},
			{Text: "cn-east-1", Description: "Suqian"},
			{Text: "cn-south-1", Description: "Guangzhou"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket", "vm", "iam-role", "bucket-acl"},
	})
}
