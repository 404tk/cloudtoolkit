package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba"
	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/oss"
)

const (
	DemoAccessKeyID     = "LTAI4tDVhjxvrWKTsEXAMPLE"
	DemoAccessKeySecret = "EXAMPLEv2fYAa2s7GhvLun7xqctKEY"
)

func ClientConfig() alibaba.ClientConfig {
	transport := newTransport()
	httpClient := &http.Client{Transport: transport}
	return alibaba.ClientConfig{
		APIOptions:          []api.Option{api.WithHTTPClient(httpClient)},
		OSSOptions:          []oss.Option{oss.WithHTTPClient(httpClient)},
		SLSHTTPClient:       httpClient,
		SkipCredentialCache: true,
	}
}
