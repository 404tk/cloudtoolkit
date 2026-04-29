package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
)

// ClientConfig builds the demo replay configuration injected into
// ucloud.NewWithConfig when replay is active for the ucloud provider.
func ClientConfig() ucloud.ClientConfig {
	transport := newTransport()
	httpClient := &http.Client{Transport: transport}
	return ucloud.ClientConfig{
		APIOptions:          []api.Option{api.WithHTTPClient(httpClient)},
		SkipCredentialCache: true,
	}
}
