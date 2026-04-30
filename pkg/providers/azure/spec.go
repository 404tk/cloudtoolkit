package azure

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/utils"
)

func init() {
	registry.Register("azure", registry.Spec{
		Options: []registry.Option{
			{Name: utils.AzureClientId, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.AzureClientSecret, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.AzureTenantId, Description: "Tenant ID", Required: true},
			{Name: utils.AzureSubscriptionId, Description: "Subscription ID"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Capabilities: []string{"cloudlist", "iam-role", "bucket-acl"},
	})
}
