package auth

import (
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Credential keeps UCloud AK/SK/token values in the same shape the console
// already uses for other providers.
type Credential struct {
	AccessKey     string
	SecretKey     string
	SecurityToken string
}

func New(accessKey, secretKey, securityToken string) Credential {
	return Credential{
		AccessKey:     accessKey,
		SecretKey:     secretKey,
		SecurityToken: securityToken,
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
	securityToken, _ := options.GetMetadata(utils.SecurityToken)
	return New(accessKey, secretKey, securityToken), nil
}

func (c Credential) Validate() error {
	switch {
	case strings.TrimSpace(c.AccessKey) == "":
		return errors.New("ucloud credential: empty access key")
	case strings.TrimSpace(c.SecretKey) == "":
		return errors.New("ucloud credential: empty secret key")
	default:
		return nil
	}
}
