package auth

import (
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Credential is the provider-local Huawei credential shape used by the
// lightweight IAM/control-plane client. Region may still be "all" at the
// provider boundary and is resolved before individual requests are sent.
type Credential struct {
	AK     string
	SK     string
	Region string
	Intl   bool
}

func New(ak, sk, region string, intl bool) Credential {
	return Credential{
		AK:     ak,
		SK:     sk,
		Region: region,
		Intl:   intl,
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
	region, _ := options.GetMetadata(utils.Region)
	version, _ := options.GetMetadata(utils.Version)
	intl := !(version == "" || version == "China")
	return New(accessKey, secretKey, region, intl), nil
}

func (c Credential) Validate() error {
	switch {
	case strings.TrimSpace(c.AK) == "":
		return errors.New("huawei credential: empty access key")
	case strings.TrimSpace(c.SK) == "":
		return errors.New("huawei credential: empty secret key")
	default:
		return nil
	}
}
