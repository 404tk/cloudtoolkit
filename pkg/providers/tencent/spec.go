package tencent

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/utils"
)

func init() {
	registry.Register("tencent", registry.Spec{
		Options: []registry.Option{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []registry.Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "ap-beijing", Description: "Beijing"},
			{Text: "ap-shanghai", Description: "Shanghai"},
			{Text: "ap-guangzhou", Description: "Guangzhou"},
			{Text: "ap-hongkong", Description: "Hong Kong"},
			{Text: "ap-singapore", Description: "Singapore"},
			{Text: "ap-seoul", Description: "Seoul"},
			{Text: "ap-tokyo", Description: "Tokyo"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket", "vm"},
	})
}
