package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/tos"
)

func ClientConfig() volcengine.ClientConfig {
	transport := newTransport()
	httpClient := &http.Client{Transport: transport}
	return volcengine.ClientConfig{
		APIOptions:          []api.Option{api.WithHTTPClient(httpClient)},
		TOSOptions:          []tos.Option{tos.WithHTTPClient(httpClient)},
		SkipCredentialCache: true,
	}
}
