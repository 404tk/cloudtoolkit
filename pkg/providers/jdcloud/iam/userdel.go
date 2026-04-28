package iam

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) DelUser() (schema.IAMResult, error) {
	ctx := context.Background()
	if d.Client == nil {
		return schema.IAMResult{}, fmt.Errorf("jdcloud iam: nil api client")
	}

	userName := strings.TrimSpace(d.UserName)
	if userName == "" {
		return schema.IAMResult{}, fmt.Errorf("empty user name")
	}

	if err := detachSubUserPolicy(ctx, d.Client, userName); err != nil && !isIgnorableDetachError(err) {
		return schema.IAMResult{}, fmt.Errorf("remove policy from %s failed: %w", userName, err)
	}
	if err := deleteSubUser(ctx, d.Client, userName); err != nil {
		return schema.IAMResult{}, fmt.Errorf("delete user %s failed: %w", userName, err)
	}

	return schema.IAMResult{
		Username: userName,
		Message:  "User deleted successfully",
	}, nil
}

func detachSubUserPolicy(ctx context.Context, client *api.Client, userName string) error {
	// JDCloud's parameter binder routes GET/DELETE/HEAD params into the query
	// string (path parameters like {subUser} are stripped). Sending policyName
	// in a JSON body here would surface as
	//   "Required request parameter 'policyName' ... is not present"
	// on the server side.
	query := url.Values{}
	query.Set("policyName", administratorPolicyName)
	var resp api.DetachSubUserPolicyResponse
	return client.DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodDelete,
		Version: "v1",
		Path:    "/subUser/" + userName + ":detachSubUserPolicy",
		Query:   query,
	}, &resp)
}

func deleteSubUser(ctx context.Context, client *api.Client, userName string) error {
	var resp api.DeleteSubUserResponse
	return client.DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodDelete,
		Version: "v1",
		Path:    "/subUser/" + userName,
	}, &resp)
}
