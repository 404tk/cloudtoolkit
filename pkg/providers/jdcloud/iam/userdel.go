package iam

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) DelUser() {
	ctx := context.Background()
	if d.Client == nil {
		logger.Error("jdcloud iam: nil api client")
		return
	}

	userName := strings.TrimSpace(d.UserName)
	if userName == "" {
		logger.Error("Empty user name.")
		return
	}

	if err := detachSubUserPolicy(ctx, d.Client, userName); err != nil && !isIgnorableDetachError(err) {
		logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", userName, err.Error()))
		return
	}
	if err := deleteSubUser(ctx, d.Client, userName); err != nil {
		logger.Error(fmt.Sprintf("Delete user %s failed: %s", userName, err.Error()))
		return
	}
	logger.Warning(userName + " user delete completed.")
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
