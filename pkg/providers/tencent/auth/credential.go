package auth

import (
	"errors"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Credential is the provider-local Tencent credential shape used by the new
// lightweight API client. It intentionally keeps only the three fields the
// project actually needs.
type Credential struct {
	SecretID  string
	SecretKey string
	Token     string
}

func New(secretID, secretKey, token string) Credential {
	return Credential{
		SecretID:  secretID,
		SecretKey: secretKey,
		Token:     token,
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
	token, _ := options.GetMetadata(utils.SecurityToken)
	return New(accessKey, secretKey, token), nil
}

func (c Credential) Validate() error {
	switch {
	case c.SecretID == "":
		return errors.New("tencent credential: empty secret id")
	case c.SecretKey == "":
		return errors.New("tencent credential: empty secret key")
	default:
		return nil
	}
}
