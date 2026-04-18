package auth

import (
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Credential is the provider-local Alibaba credential shape used by the new
// lightweight RPC client. It keeps only the fields the project actually needs.
type Credential struct {
	AccessKeyID     string
	AccessKeySecret string
	SecurityToken   string
}

func New(accessKeyID, accessKeySecret, securityToken string) Credential {
	return Credential{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		SecurityToken:   securityToken,
	}
}

func FromOptions(options schema.Options) (Credential, error) {
	accessKeyID, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return Credential{}, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	accessKeySecret, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return Credential{}, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}
	securityToken, _ := options.GetMetadata(utils.SecurityToken)
	return New(accessKeyID, accessKeySecret, securityToken), nil
}

func (c Credential) Validate() error {
	switch {
	case strings.TrimSpace(c.AccessKeyID) == "":
		return errors.New("alibaba credential: empty access key id")
	case strings.TrimSpace(c.AccessKeySecret) == "":
		return errors.New("alibaba credential: empty access key secret")
	default:
		return nil
	}
}
