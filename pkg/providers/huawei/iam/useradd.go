package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) AddUser() {
	ctx := context.Background()
	uid, domainID, err := d.createUser(ctx)
	if err != nil {
		logger.Error("Create user failed:", err.Error())
		return
	}
	err = d.addUserToAdminGroup(ctx, uid)
	if err != nil {
		logger.Error("Grant AdministratorAccess policy failed.")
		return
	}
	name := d.getDomainName(ctx, domainID)
	fmt.Printf("\n%-10s\t%-10s\t%-60s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-10s\t%-10s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-10s\t%-60s\n\n",
		d.Username,
		d.Password, "https://auth.huaweicloud.com/authui/login?id="+name)
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
		logger.Error("List domains failed:", err.Error())
		return ""
	}
	for _, v := range resp.Domains {
		if v.ID == domainID {
			return v.Name
		}
	}
	return ""
}
