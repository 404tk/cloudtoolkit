package replay

import demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"

const (
	demoMasterPin     = "ctk-demo-master"
	demoMasterAccount = demoMasterPin + "@ctk.demo"
	demoRegion        = "cn-north-1"
)

var demoCredentials = loadDemoCredentials()

func loadDemoCredentials() demoreplay.Credentials {
	creds, ok := demoreplay.CredentialsFor("jdcloud")
	if !ok {
		return demoreplay.Credentials{}
	}
	return creds
}
