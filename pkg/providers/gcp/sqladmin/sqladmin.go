package sqladmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// Driver wraps Cloud SQL Admin user operations used by rds-account-check.
// `instanceID` is the Cloud SQL instance name (without project prefix); the
// driver picks the project from the credential's `Projects` slice.
type Driver struct {
	Client   *api.Client
	Projects []string
}

// CreateAccount provisions a Cloud SQL user. Username/password come from the
// `rds-account-check` config. Host is left blank (Cloud SQL semantics: empty
// host == any host for MySQL; ignored for Postgres).
func (d *Driver) CreateAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil || d.Client == nil {
		return schema.DatabaseActionResult{}, errors.New("gcp sqladmin: nil api client")
	}
	project, err := d.project()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	user, password, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	body, err := json.Marshal(api.SQLUser{
		Name:     user,
		Password: password,
		Project:  project,
		Instance: instanceID,
	})
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	var op api.SQLOperation
	err = d.Client.Do(ctx, api.Request{
		Method:  http.MethodPost,
		BaseURL: api.SQLAdminBaseURL,
		Path:    fmt.Sprintf("/sql/v1beta4/projects/%s/instances/%s/users", url.PathEscape(project), url.PathEscape(instanceID)),
		Body:    body,
	}, &op)
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "useradd",
		Username: user,
		Password: password,
		Message:  fmt.Sprintf("Cloud SQL user created on %s", instanceID),
	}, nil
}

// DeleteAccount removes the Cloud SQL user named by `rds-account-check` from
// instanceID.
func (d *Driver) DeleteAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil || d.Client == nil {
		return schema.DatabaseActionResult{}, errors.New("gcp sqladmin: nil api client")
	}
	project, err := d.project()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	user, _, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	query := url.Values{}
	query.Set("name", user)
	var op api.SQLOperation
	err = d.Client.Do(ctx, api.Request{
		Method:  http.MethodDelete,
		BaseURL: api.SQLAdminBaseURL,
		Path:    fmt.Sprintf("/sql/v1beta4/projects/%s/instances/%s/users", url.PathEscape(project), url.PathEscape(instanceID)),
		Query:   query,
	}, &op)
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "userdel",
		Username: user,
		Message:  user + " account delete completed.",
	}, nil
}

func (d *Driver) project() (string, error) {
	for _, p := range d.Projects {
		p = strings.TrimSpace(p)
		if p != "" {
			return p, nil
		}
	}
	return "", errors.New("gcp sqladmin: no project configured")
}

func parseRDSAccount() (string, string, error) {
	accountName, accountPassword, ok := strings.Cut(env.Active().RDSAccount, ":")
	if !ok || strings.TrimSpace(accountName) == "" || strings.TrimSpace(accountPassword) == "" {
		return "", "", fmt.Errorf("RDS account metadata is invalid: expected username:password")
	}
	return accountName, accountPassword, nil
}
