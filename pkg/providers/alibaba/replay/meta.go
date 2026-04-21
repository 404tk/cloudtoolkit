package replay

import "strings"

type Credentials struct {
	AccessKey string
	SecretKey string
}

func SupportsProvider(provider string) bool {
	return normalizeProvider(provider) == "alibaba"
}

func CredentialsFor(provider string) (Credentials, bool) {
	if !SupportsProvider(provider) {
		return Credentials{}, false
	}
	return Credentials{
		AccessKey: DemoAccessKeyID,
		SecretKey: DemoAccessKeySecret,
	}, true
}

func normalizeProvider(provider string) string {
	return strings.TrimSpace(provider)
}
