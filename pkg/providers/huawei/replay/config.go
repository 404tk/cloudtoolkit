package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/obs"
)

// ClientConfig builds the demo replay configuration injected into
// huawei.NewWithConfig when replay is active for the huawei provider.
func ClientConfig() huawei.ClientConfig {
	transport := newTransport()
	httpClient := &http.Client{Transport: transport}
	return huawei.ClientConfig{
		APIOptions:          []api.Option{api.WithHTTPClient(httpClient)},
		OBSOptions:          []obs.Option{obs.WithHTTPClient(httpClient)},
		SkipCredentialCache: true,
	}
}
