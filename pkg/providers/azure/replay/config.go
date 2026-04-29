package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/azure"
)

// ClientConfig builds the demo replay configuration injected into
// azure.NewWithConfig when replay is active for the azure provider.
func ClientConfig() azure.ClientConfig {
	transport := newTransport()
	return azure.ClientConfig{
		HTTPClient:          &http.Client{Transport: transport},
		SkipCredentialCache: true,
	}
}
