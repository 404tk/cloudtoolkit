package replay

import demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"

const (
	demoProjectID  = "ctk-demo-project"
	demoAccessToken = "ctk.demo.gcp.bearer"
	demoZone       = "us-central1-a"
)

var demoCredentials = loadDemoCredentials()

func loadDemoCredentials() demoreplay.Credentials {
	creds, ok := demoreplay.CredentialsFor("gcp")
	if !ok {
		return demoreplay.Credentials{}
	}
	return creds
}
