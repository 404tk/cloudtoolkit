package auth

import (
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Credential is the provider-local JDCloud AK/SK shape used by the
// lightweight REST client.
type Credential struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
}

func New(accessKey, secretKey, sessionToken string) Credential {
	return Credential{
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		SessionToken: sessionToken,
	}
}

func FromOptions(options schema.Options) (Credential, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return Credential{}, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return Credential{}, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}
	sessionToken, _ := options.GetMetadata(utils.SecurityToken)
	return New(accessKey, secretKey, sessionToken), nil
}

func (c Credential) Validate() error {
	switch {
	case strings.TrimSpace(c.AccessKey) == "":
		return errors.New("jdcloud credential: empty access key")
	case strings.TrimSpace(c.SecretKey) == "":
		return errors.New("jdcloud credential: empty secret key")
	default:
		return nil
	}
}
