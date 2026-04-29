package replay

import (
	"strings"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type Credentials struct {
	AccessKey string
	SecretKey string
}

func SupportsProvider(provider string) bool {
	return strings.TrimSpace(provider) == "aws"
}

func CredentialsFor(provider string) (Credentials, bool) {
	if !SupportsProvider(provider) {
		return Credentials{}, false
	}
	creds, ok := demoreplay.CredentialsFor("aws")
	if !ok {
		return Credentials{}, false
	}
	return Credentials{
		AccessKey: creds.AccessKey,
		SecretKey: creds.SecretKey,
	}, true
}
