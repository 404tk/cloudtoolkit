package cloud

import "github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"

// Endpoints holds the ARM-relevant hostnames for one national cloud.
type Endpoints struct {
	ActiveDirectory string
	ResourceManager string
	TokenAudience   string
	Name            string
}

func For(c auth.Cloud) Endpoints {
	normalized := auth.New("", "", "", "", c).Cloud
	return Endpoints{
		ActiveDirectory: normalized.ActiveDirectoryEndpoint(),
		ResourceManager: normalized.ResourceManagerEndpoint(),
		TokenAudience:   normalized.ResourceManagerEndpoint(),
		Name:            string(normalized),
	}
}
