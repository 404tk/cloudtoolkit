package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp"
)

// ClientConfig builds the demo replay configuration injected into
// gcp.NewWithConfig when replay is active for the gcp provider.
func ClientConfig() gcp.ClientConfig {
	transport := newTransport()
	return gcp.ClientConfig{
		HTTPClient:          &http.Client{Transport: transport},
		SkipCredentialCache: true,
	}
}
