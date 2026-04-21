package auth

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

func TestFromOptions(t *testing.T) {
	payload, err := json.Marshal(map[string]string{
		"type":           "service_account",
		"project_id":     "demo-project",
		"private_key_id": "kid-1",
		"private_key":    testutil.PKCS8PrivateKeyPEM,
		"client_email":   "demo@example.com",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	options := schema.Options{
		utils.GCPserviceAccountJSON: base64.StdEncoding.EncodeToString(payload),
	}

	cred, err := FromOptions(options)
	if err != nil {
		t.Fatalf("FromOptions() error = %v", err)
	}
	if cred.ProjectID != "demo-project" || cred.ClientEmail != "demo@example.com" {
		t.Fatalf("unexpected credential: %+v", cred)
	}
	if cred.TokenURI != DefaultTokenURI {
		t.Fatalf("unexpected token uri: %s", cred.TokenURI)
	}
	if len(cred.Scopes) != 1 || cred.Scopes[0] != DefaultScope {
		t.Fatalf("unexpected scopes: %v", cred.Scopes)
	}
}

func TestFromOptionsRejectsUnsupportedCredentialType(t *testing.T) {
	options := schema.Options{
		utils.GCPserviceAccountJSON: base64.StdEncoding.EncodeToString([]byte(`{
			"type":"authorized_user",
			"project_id":"demo-project",
			"private_key":"ignored",
			"client_email":"demo@example.com"
		}`)),
	}

	if _, err := FromOptions(options); err == nil || err.Error() != "gcp: only service_account credentials are supported" {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}

func TestCredentialValidate(t *testing.T) {
	cred := Credential{
		ProjectID:     "demo-project",
		PrivateKeyPEM: testutil.PKCS8PrivateKeyPEM,
		ClientEmail:   "demo@example.com",
		TokenURI:      DefaultTokenURI,
		Scopes:        []string{DefaultScope},
	}
	if err := cred.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
