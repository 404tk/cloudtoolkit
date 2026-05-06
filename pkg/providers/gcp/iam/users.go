package iam

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// EnableServiceAccount enables (i.e. unlocks) a GCP service account.
// `principal` is the service account email or its short name; the project is
// taken from the credential. This is the closest CSPM-detectable
// `useradd`-style lever GCP exposes via API — there is no Cloud Identity
// "create user" without a paid Google Workspace tenant.
func (d *Driver) EnableServiceAccount(ctx context.Context, principal string) error {
	if d == nil || d.Client == nil {
		return errors.New("gcp iam: nil api client")
	}
	resourceID, err := d.serviceAccountResourceID(principal)
	if err != nil {
		return err
	}
	return d.Client.Do(ctx, api.Request{
		Method:  http.MethodPost,
		BaseURL: api.IAMBaseURL,
		Path:    "/v1/" + resourceID + ":enable",
		Body:    []byte("{}"),
	}, nil)
}

// DisableServiceAccount disables a GCP service account, revoking the access
// granted by `EnableServiceAccount`.
func (d *Driver) DisableServiceAccount(ctx context.Context, principal string) error {
	if d == nil || d.Client == nil {
		return errors.New("gcp iam: nil api client")
	}
	resourceID, err := d.serviceAccountResourceID(principal)
	if err != nil {
		return err
	}
	return d.Client.Do(ctx, api.Request{
		Method:  http.MethodPost,
		BaseURL: api.IAMBaseURL,
		Path:    "/v1/" + resourceID + ":disable",
		Body:    []byte("{}"),
	}, nil)
}

// AddUser is a thin schema.IAMResult-shaped wrapper around
// EnableServiceAccount used by the iam-user-check `add` action.
func (d *Driver) AddUser(ctx context.Context, principal string) (schema.IAMResult, error) {
	if err := d.EnableServiceAccount(ctx, principal); err != nil {
		return schema.IAMResult{}, err
	}
	return schema.IAMResult{
		Action:   "add",
		Username: principal,
		Message:  "service account enabled",
	}, nil
}

// DelUser is a thin schema.IAMResult-shaped wrapper around
// DisableServiceAccount used by the iam-user-check `del` action.
func (d *Driver) DelUser(ctx context.Context, principal string) (schema.IAMResult, error) {
	if err := d.DisableServiceAccount(ctx, principal); err != nil {
		return schema.IAMResult{}, err
	}
	return schema.IAMResult{
		Action:   "del",
		Username: principal,
		Message:  "service account disabled",
	}, nil
}

func (d *Driver) serviceAccountResourceID(principal string) (string, error) {
	principal = strings.TrimSpace(principal)
	if principal == "" {
		return "", errors.New("gcp iam: principal (service account email or unique id) required")
	}
	if strings.HasPrefix(principal, "projects/") {
		return principal, nil
	}
	if len(d.Projects) == 0 || strings.TrimSpace(d.Projects[0]) == "" {
		return "", errors.New("gcp iam: no project configured")
	}
	return fmt.Sprintf("projects/%s/serviceAccounts/%s", url.PathEscape(d.Projects[0]), url.PathEscape(principal)), nil
}
