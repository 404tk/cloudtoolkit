package ucloud

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/utils"
)

func init() {
	registry.Register("ucloud", registry.Spec{
		Options: []registry.Option{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.ProjectID, Description: "Project ID"},
		},
		Regions: []registry.Suggestion{
			{Text: "all", Description: "enumerate all accessible regions"},
			{Text: "cn-bj2", Description: "Beijing"},
			{Text: "cn-sh2", Description: "Shanghai"},
			{Text: "cn-gd", Description: "Guangzhou"},
			{Text: "hk", Description: "Hong Kong"},
			{Text: "sg", Description: "Singapore"},
			{Text: "us-ca", Description: "Los Angeles"},
			{Text: "th-bkk", Description: "Bangkok"},
			{Text: "ge-fra", Description: "Frankfurt"},
		},
		Capabilities: []string{"cloudlist", "iam", "iam-role", "bucket", "bucket-acl"},
	})
}
