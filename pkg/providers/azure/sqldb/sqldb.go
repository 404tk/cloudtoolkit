// Package sqldb wraps the Azure SQL master password rotation. Azure SQL
// has no native "user" management via ARM (T-SQL is the only way to
// create login/users); the CSPM-detectable management-plane signal is
// `Servers - Update` rotating administratorLoginPassword.
package sqldb

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Driver struct {
	Client          *azapi.Client
	SubscriptionIDs []string
}

// CreateAccount rotates the SQL server administrator password to the value
// supplied by the `rds-account-check` config. instanceID is parsed as
// `<resourceGroup>/<serverName>`.
func (d *Driver) CreateAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil || d.Client == nil {
		return schema.DatabaseActionResult{}, errors.New("azure sqldb: nil api client")
	}
	subscription, err := d.subscription()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	resourceGroup, server, err := splitInstanceID(instanceID)
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	user, password, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	if err := d.patchPassword(ctx, subscription, resourceGroup, server, password); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:    "useradd",
		Username:  user,
		Password:  password,
		Privilege: "AdministratorLogin",
		Message:   fmt.Sprintf("Azure SQL administrator password rotated to known value on %s/%s", resourceGroup, server),
	}, nil
}

// DeleteAccount rotates the SQL server administrator password to a random
// value, revoking access via the known credential.
func (d *Driver) DeleteAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil || d.Client == nil {
		return schema.DatabaseActionResult{}, errors.New("azure sqldb: nil api client")
	}
	subscription, err := d.subscription()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	resourceGroup, server, err := splitInstanceID(instanceID)
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	password, err := randomPassword()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	if err := d.patchPassword(ctx, subscription, resourceGroup, server, password); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:  "userdel",
		Message: fmt.Sprintf("Azure SQL administrator password rotated to random value on %s/%s; access via known credential revoked", resourceGroup, server),
	}, nil
}

func (d *Driver) patchPassword(ctx context.Context, subscription, resourceGroup, server, password string) error {
	body, err := json.Marshal(azapi.SQLServerPatch{
		Properties: azapi.SQLServerProperties{AdministratorLoginPassword: password},
	})
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Set("api-version", azapi.SQLAPIVersion)
	return d.Client.Do(ctx, azapi.Request{
		Method: http.MethodPatch,
		Path: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Sql/servers/%s",
			url.PathEscape(subscription), url.PathEscape(resourceGroup), url.PathEscape(server)),
		Query: query,
		Body:  body,
	}, nil)
}

func (d *Driver) subscription() (string, error) {
	for _, sub := range d.SubscriptionIDs {
		sub = strings.TrimSpace(sub)
		if sub != "" {
			return sub, nil
		}
	}
	return "", errors.New("azure sqldb: no subscription configured")
}

func splitInstanceID(instanceID string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(instanceID), "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("azure sqldb: instance id must be `<resourceGroup>/<server>`, got %q", instanceID)
	}
	return parts[0], parts[1], nil
}

func parseRDSAccount() (string, string, error) {
	accountName, accountPassword, ok := strings.Cut(env.Active().RDSAccount, ":")
	if !ok || strings.TrimSpace(accountName) == "" || strings.TrimSpace(accountPassword) == "" {
		return "", "", fmt.Errorf("RDS account metadata is invalid: expected username:password")
	}
	return accountName, accountPassword, nil
}

func randomPassword() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.ReplaceAll(base64.StdEncoding.EncodeToString(buf), "/", "_"), nil
}
