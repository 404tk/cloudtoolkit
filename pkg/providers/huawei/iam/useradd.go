package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) AddUser() (schema.IAMResult, error) {
	ctx := context.Background()
	uid, domainID, err := d.createUser(ctx)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("create user failed: %w", err)
	}
	err = d.addUserToAdminGroup(ctx, uid)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("grant AdministratorAccess policy failed: %w", err)
	}
	name := d.getDomainName(ctx, domainID)
	loginURL := "https://auth.huaweicloud.com/authui/login?id=" + name

	return schema.IAMResult{
		Username:  d.Username,
		Password:  d.Password,
		LoginURL:  loginURL,
		AccountID: name,
		Message:   "User created successfully with admin group membership",
	}, nil
}

func (d *Driver) createUser(ctx context.Context) (string, string, error) {
	region, err := d.requestRegion()
	if err != nil {
		return "", "", err
	}
	domainID := d.resolveDomainID(ctx)
	body, err := json.Marshal(api.CreateUserRequest{
		User: api.CreateUserOption{
			Name:     d.Username,
			Password: d.Password,
			Enabled:  true,
			DomainID: domainID,
		},
	})
	if err != nil {
		return "", "", err
	}

	var resp api.CreateUserResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  region,
		Intl:    d.Cred.Intl,
		Method:  http.MethodPost,
		Path:    "/v3/users",
		Body:    body,
		Headers: d.domainHeaders(ctx),
	}, &resp); err != nil {
		return "", "", err
	}
	if resp.User.DomainID != "" {
		d.DomainID = resp.User.DomainID
	}
	return resp.User.ID, resp.User.DomainID, nil
}

func (d *Driver) addUserToAdminGroup(ctx context.Context, uid string) error {
	region, err := d.requestRegion()
	if err != nil {
		return err
	}
	var resp api.ListGroupsResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "iam",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v3/groups",
		Idempotent: true,
		Headers:    d.domainHeaders(ctx),
	}, &resp); err != nil {
		return err
	}

	groups := make(map[string]string)
	for _, v := range resp.Groups {
		groups[v.Name] = v.ID
	}

	if g, ok := groups["admin"]; ok {
		return d.addUserToGroup(ctx, region, g, uid)
	}
	for _, g := range groups {
		if err := d.addUserToGroup(ctx, region, g, uid); err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) addUserToGroup(ctx context.Context, region, groupID, userID string) error {
	return d.client().DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  region,
		Intl:    d.Cred.Intl,
		Method:  http.MethodPut,
		Path:    fmt.Sprintf("/v3/groups/%s/users/%s", groupID, userID),
		Headers: d.domainHeaders(ctx),
	}, nil)
}

func (d *Driver) getDomainName(ctx context.Context, domainID string) string {
	resp, err := d.listAuthDomains(ctx)
	if err != nil {
		return ""
	}
	for _, v := range resp.Domains {
		if v.ID == domainID {
			return v.Name
		}
	}
	return ""
}
