package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
)

// ClientConfig builds the demo replay configuration injected into
// jdcloud.NewWithConfig when replay is active for the jdcloud provider.
func ClientConfig() jdcloud.ClientConfig {
	transport := newTransport()
	httpClient := &http.Client{Transport: transport}
	return jdcloud.ClientConfig{
		APIOptions:          []api.Option{api.WithHTTPClient(httpClient)},
		SkipCredentialCache: true,
	}
}
