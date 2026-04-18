package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
)

// Credential is the provider-local AWS credential shape used by the
// lightweight STS client and direct sdk.Config construction.
type Credential struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

func New(accessKeyID, secretAccessKey, sessionToken string) Credential {
	return Credential{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
	}
}

func FromOptions(options schema.Options) (Credential, error) {
	accessKeyID, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return Credential{}, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretAccessKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return Credential{}, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}
	sessionToken, _ := options.GetMetadata(utils.SecurityToken)
	return New(accessKeyID, secretAccessKey, sessionToken), nil
}

func (c Credential) Validate() error {
	switch {
	case strings.TrimSpace(c.AccessKeyID) == "":
		return errors.New("aws credential: empty access key id")
	case strings.TrimSpace(c.SecretAccessKey) == "":
		return errors.New("aws credential: empty secret access key")
	default:
		return nil
	}
}

func (c Credential) Retrieve(context.Context) (awsv2.Credentials, error) {
	if err := c.Validate(); err != nil {
		return awsv2.Credentials{}, err
	}
	return awsv2.Credentials{
		AccessKeyID:     c.AccessKeyID,
		SecretAccessKey: c.SecretAccessKey,
		SessionToken:    c.SessionToken,
		Source:          "ctk-static",
	}, nil
}
