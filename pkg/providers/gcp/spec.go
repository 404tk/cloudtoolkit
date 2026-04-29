package gcp

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/registry"
	"github.com/404tk/cloudtoolkit/utils"
)

func init() {
	registry.Register("gcp", registry.Spec{
		Options: []registry.Option{
			{Name: utils.GCPserviceAccountJSON, Description: "GCP Credential encoded through Base64", Required: true, Sensitive: true},
		},
		Capabilities: []string{"cloudlist"},
	})
}
