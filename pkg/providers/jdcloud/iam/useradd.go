package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const (
	// administratorPolicyName is JDCloud's built-in system policy that grants
	// full administrative privilege to a sub user.
	administratorPolicyName = "JDCloudAdmin-New"

	// jdcloudConsoleURL is the public console entry for JDCloud sub users.
	// Sub users sign in with {masterPin}@{subUserName} or just the sub user
	// account id, depending on how the master account enabled console login.
	jdcloudConsoleURL = "https://login.jdcloud.com/subAccount/login/%s"
)

func (d *Driver) AddUser() (schema.IAMResult, error) {
	ctx := context.Background()
	if d.Client == nil {
		return schema.IAMResult{}, fmt.Errorf("jdcloud iam: nil api client")
	}

	userName := strings.TrimSpace(d.UserName)
	password := d.Password
	if userName == "" {
		return schema.IAMResult{}, fmt.Errorf("empty user name")
	}
	if password == "" {
		return schema.IAMResult{}, fmt.Errorf("empty password")
	}

	if err := createSubUser(ctx, d.Client, userName, password); err != nil {
		return schema.IAMResult{}, fmt.Errorf("create user failed: %w", err)
	}
	if err := attachSubUserPolicy(ctx, d.Client, userName); err != nil {
		return schema.IAMResult{}, fmt.Errorf("grant %s policy failed: %w", administratorPolicyName, err)
	}

	masterPin := getMasterPin(ctx, d.Client, d.AccessKey)
	loginURL := fmt.Sprintf(jdcloudConsoleURL, masterPin)

	return schema.IAMResult{
		Username:  userName,
		Password:  password,
		LoginURL:  loginURL,
		AccountID: masterPin,
		Message:   "User created successfully with " + administratorPolicyName + " policy",
	}, nil
}

func createSubUser(ctx context.Context, client *api.Client, userName, password string) error {
	consoleLogin := true
	createAk := false
	needReset := false
	body, err := json.Marshal(api.CreateSubUserRequest{
		CreateSubUserInfo: api.CreateSubUserInfo{
			Name:              userName,
			Password:          password,
			ConsoleLogin:      &consoleLogin,
			CreateAk:          &createAk,
			NeedResetPassword: &needReset,
		},
	})
	if err != nil {
		return err
	}
	var resp api.CreateSubUserResponse
	return client.DoJSON(ctx, api.Request{
		Service: "iam",
		// IAM is global; let the signer fall back to the jdcloud-api scope.
		Region:  "",
		Method:  http.MethodPost,
		Version: "v1",
		Path:    "/subUser",
		Body:    body,
	}, &resp)
}

func describeUserPin(ctx context.Context, client *api.Client, accessKey string) (string, error) {
	accessKey = strings.TrimSpace(accessKey)
	if accessKey == "" {
		return "", errors.New("jdcloud iam: empty access key")
	}

	// The IAM DescribeUserPin path is /regions/{regionId}/user:describeUserPin.
	// The service itself is global (signing region still falls back to
	// "jdcloud-api"), but the path placeholder needs a syntactically valid
	// region; cn-north-1 is the fleet-wide default other drivers already use.
	const probeRegion = "cn-north-1"
	query := url.Values{}
	query.Set("accessKey", accessKey)

	var resp api.DescribeUserPinResponse
	err := client.DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodGet,
		Version: "v1",
		Path:    "/regions/" + probeRegion + "/user:describeUserPin",
		Query:   query,
	}, &resp)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Result.Pin), nil
}

// getMasterPin resolves the authenticated principal's pin via IAM
// DescribeUserPin so the sub-user login URL can be printed with the master
// account context pre-filled. JDCloud's API requires either accessKey or
// accountId; we use the authenticated master access key that created the
// sub-user. Failure is non-fatal — the user creation itself has already
// succeeded, we just skip the pre-filled URL parameter.
func getMasterPin(ctx context.Context, client *api.Client, accessKey string) string {
	pin, err := describeUserPin(ctx, client, accessKey)
	if err != nil {
		return ""
	}
	return pin
}

func attachSubUserPolicy(ctx context.Context, client *api.Client, userName string) error {
	body, err := json.Marshal(api.AttachSubUserPolicyRequest{
		SubUser:    userName,
		PolicyName: administratorPolicyName,
	})
	if err != nil {
		return err
	}
	var resp api.AttachSubUserPolicyResponse
	return client.DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodPost,
		Version: "v1",
		Path:    "/subUser/" + userName + ":attachSubUserPolicy",
		Body:    body,
	}, &resp)
}

// jdcloudResourceNotFoundCode is returned by JDCloud IAM for "resource does
// not exist / policy does not exist" on detach/delete paths. The error message
// is localised (Chinese), so matching on the code is more reliable than
// substring-scanning the message.
const jdcloudResourceNotFoundCode = 1011

// isIgnorableDetachError returns true for the idempotent tail of detach/delete
// flows: policy attachment already missing, sub user already gone. JDCloud's
// message locale follows the caller's IP/account region (often Chinese), so
// we lean on the HTTP status and the numeric error code first and only use
// substring matching as a last resort.
func isIgnorableDetachError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		if apiErr.HTTPStatus == http.StatusNotFound {
			return true
		}
		if apiErr.Code == jdcloudResourceNotFoundCode {
			return true
		}
		msg := strings.ToLower(apiErr.Message)
		if strings.Contains(msg, "not exist") ||
			strings.Contains(msg, "not found") ||
			strings.Contains(msg, "already detached") ||
			strings.Contains(msg, "no such") ||
			// Chinese locale fall-throughs; JDCloud returns "资源不存在" /
			// "策略不存在" for missing resources without a code we can match.
			strings.Contains(apiErr.Message, "不存在") {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not exist") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "already detached")
}
