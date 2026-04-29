package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
)

const (
	DemoAccessKeyID     = "AKIAIOSFODNN7EXAMPLE"
	DemoAccessKeySecret = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
)

func ClientConfig() aws.ClientConfig {
	transport := newTransport()
	httpClient := &http.Client{Transport: transport}
	return aws.ClientConfig{
		APIOptions:          []api.Option{api.WithHTTPClient(httpClient)},
		SkipCredentialCache: true,
	}
}
