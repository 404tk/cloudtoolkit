package replay

import demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"

const (
	demoTenantID       = "11111111-2222-3333-4444-555555555555"
	demoSubscriptionID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	demoSubscriptionDN = "ctk-validation"
	demoAccessToken    = "ctk.demo.bearer.replay"
	demoResourceGroup  = "ctk-demo-rg"
	demoLocation       = "eastus"
)

var demoCredentials = loadDemoCredentials()

func loadDemoCredentials() demoreplay.Credentials {
	creds, ok := demoreplay.CredentialsFor("azure")
	if !ok {
		return demoreplay.Credentials{}
	}
	return creds
}
