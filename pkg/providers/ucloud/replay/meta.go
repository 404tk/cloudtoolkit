package replay

import demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"

const (
	demoCompanyID   = int64(900001)
	demoUserID      = 9001
	demoUserName    = "ctk-demo-admin"
	demoUserEmail   = "ctk-demo@validation.local"
	demoProjectID   = "org-ctkdemo"
	demoProjectName = "ctk-validation"
)

var demoCredentials = loadDemoCredentials()

func loadDemoCredentials() demoreplay.Credentials {
	creds, ok := demoreplay.CredentialsFor("ucloud")
	if !ok {
		return demoreplay.Credentials{}
	}
	return creds
}
