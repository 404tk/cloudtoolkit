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
	return strings.TrimSpace(provider) == "alibaba"
}

func CredentialsFor(provider string) (Credentials, bool) {
	if !SupportsProvider(provider) {
		return Credentials{}, false
	}
	creds, ok := demoreplay.CredentialsFor("alibaba")
	if !ok {
		return Credentials{}, false
	}
	return Credentials{
		AccessKey: creds.AccessKey,
		SecretKey: creds.SecretKey,
	}, true
}
