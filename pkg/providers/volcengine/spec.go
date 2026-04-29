package volcengine

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/utils"
)

func init() {
	registry.Register("volcengine", registry.Spec{
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
			{Text: "cn-shanghai", Description: "Shanghai"},
			{Text: "ap-southeast-1", Description: "Singapore"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket", "vm"},
	})
}
