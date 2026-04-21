package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Credential is the provider-local GCP credential shape carved out of the
// service account JSON.
type Credential struct {
	Type          string
	ProjectID     string
	PrivateKeyID  string
	PrivateKeyPEM string
	ClientEmail   string
	TokenURI      string
	Scopes        []string
}

const DefaultTokenURI = "https://oauth2.googleapis.com/token"
const DefaultScope = "https://www.googleapis.com/auth/cloud-platform"

type serviceAccountJSON struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	TokenURI     string `json:"token_uri"`
}

func FromOptions(options schema.Options) (Credential, error) {
	value, ok := options.GetMetadata(utils.GCPserviceAccountJSON)
	if !ok {
		return Credential{}, &schema.ErrNoSuchKey{Name: utils.GCPserviceAccountJSON}
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return Credential{}, err
	}

	var payload serviceAccountJSON
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return Credential{}, err
	}

	if payload.Type != "service_account" {
		return Credential{}, errors.New("gcp: only service_account credentials are supported")
	}

	tokenURI := strings.TrimSpace(payload.TokenURI)
	if tokenURI == "" {
		tokenURI = DefaultTokenURI
	}

	return Credential{
		Type:          payload.Type,
		ProjectID:     strings.TrimSpace(payload.ProjectID),
		PrivateKeyID:  strings.TrimSpace(payload.PrivateKeyID),
		PrivateKeyPEM: payload.PrivateKey,
		ClientEmail:   strings.TrimSpace(payload.ClientEmail),
		TokenURI:      tokenURI,
		Scopes:        []string{DefaultScope},
	}, nil
}

func (c Credential) Validate() error {
	switch {
	case strings.TrimSpace(c.ProjectID) == "":
		return errors.New("gcp credential: empty project id")
	case strings.TrimSpace(c.PrivateKeyPEM) == "":
		return errors.New("gcp credential: empty private key")
	case strings.TrimSpace(c.ClientEmail) == "":
		return errors.New("gcp credential: empty client email")
	case strings.TrimSpace(c.TokenURI) == "":
		return errors.New("gcp credential: empty token uri")
	case len(c.Scopes) == 0:
		return errors.New("gcp credential: empty scopes")
	default:
		return nil
	}
}
