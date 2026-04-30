package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
)

// ListKeys enumerates the keys for a service account. accountID may be an
// email or "projects/{p}/serviceAccounts/{email}" form.
func (d *Driver) ListKeys(ctx context.Context, project, accountID string) ([]api.ServiceAccountKey, error) {
	if d == nil || d.Client == nil {
		return nil, fmt.Errorf("gcp iam: nil client")
	}
	resourceID := serviceAccountResourceID(project, accountID)
	var resp api.ListServiceAccountKeysResponse
	if err := d.Client.Do(ctx, api.Request{
		Method:     http.MethodGet,
		BaseURL:    api.IAMBaseURL,
		Path:       "/v1/" + resourceID + "/keys",
		Idempotent: true,
	}, &resp); err != nil {
		return nil, err
	}
	return resp.Keys, nil
}

// CreateKey mints a new user-managed key for a service account. The
// returned ServiceAccountKey carries PrivateKeyData (base64 of the JSON
// credential file) which is only ever returned once.
func (d *Driver) CreateKey(ctx context.Context, project, accountID string) (api.ServiceAccountKey, error) {
	if d == nil || d.Client == nil {
		return api.ServiceAccountKey{}, fmt.Errorf("gcp iam: nil client")
	}
	resourceID := serviceAccountResourceID(project, accountID)
	body, err := json.Marshal(api.CreateServiceAccountKeyRequest{})
	if err != nil {
		return api.ServiceAccountKey{}, err
	}
	var key api.ServiceAccountKey
	if err := d.Client.Do(ctx, api.Request{
		Method:  http.MethodPost,
		BaseURL: api.IAMBaseURL,
		Path:    "/v1/" + resourceID + "/keys",
		Body:    body,
	}, &key); err != nil {
		return api.ServiceAccountKey{}, err
	}
	return key, nil
}

// DeleteKey revokes a service-account key by ID. keyID may be the trailing
// segment or the full "projects/.../keys/{id}" form.
func (d *Driver) DeleteKey(ctx context.Context, project, accountID, keyID string) error {
	if d == nil || d.Client == nil {
		return fmt.Errorf("gcp iam: nil client")
	}
	resourceID := serviceAccountResourceID(project, accountID)
	id := keyResourceID(resourceID, keyID)
	return d.Client.Do(ctx, api.Request{
		Method:  http.MethodDelete,
		BaseURL: api.IAMBaseURL,
		Path:    "/v1/" + id,
	}, nil)
}

// serviceAccountResourceID returns the canonical
// "projects/{p}/serviceAccounts/{email}" identifier for a service account.
// accountID may already be in that form, or just an email.
func serviceAccountResourceID(project, accountID string) string {
	accountID = strings.TrimSpace(accountID)
	if strings.HasPrefix(accountID, "projects/") {
		return accountID
	}
	return "projects/" + url.PathEscape(project) + "/serviceAccounts/" + url.PathEscape(accountID)
}

// keyResourceID joins a key id to a service account resource path. keyID may
// already be the full path or just the trailing segment.
func keyResourceID(serviceAccount, keyID string) string {
	keyID = strings.TrimSpace(keyID)
	if strings.HasPrefix(keyID, "projects/") {
		return keyID
	}
	return serviceAccount + "/keys/" + keyID
}

// KeyShortID extracts the trailing segment from a key resource name like
// "projects/.../keys/<id>". Returns the input unchanged if it has no slash.
func KeyShortID(name string) string {
	idx := strings.LastIndex(name, "/")
	if idx < 0 {
		return name
	}
	return name[idx+1:]
}
