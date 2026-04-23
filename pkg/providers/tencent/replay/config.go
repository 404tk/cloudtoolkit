package replay

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cos"
)

func ClientConfig() tencent.ClientConfig {
	httpClient := replayHTTPClient()
	return tencent.ClientConfig{
		APIOptions:          []api.Option{api.WithHTTPClient(httpClient)},
		COSOptions:          []cos.Option{cos.WithHTTPClient(httpClient)},
		SkipCredentialCache: true,
	}
}
