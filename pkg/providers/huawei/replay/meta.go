package replay

import demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"

const (
	demoDomainName = "ctk-demo"
	demoDomainID   = "06f1d2dca680f0a02fa4c01acc0e0001"
	demoUserID     = "06f1d2dca680f0a02fa4c01acc0e0099"
	demoUserName   = "ctk-demo-admin"
	defaultRegion  = "cn-north-4"
)

func loadDemoCredentials() demoreplay.Credentials {
	creds, ok := demoreplay.CredentialsFor("huawei")
	if !ok {
		return demoreplay.Credentials{}
	}
	return creds
}

var demoCredentials = loadDemoCredentials()
