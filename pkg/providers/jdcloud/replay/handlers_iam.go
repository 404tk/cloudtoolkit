package replay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleIAM(req *http.Request, body []byte) (*http.Response, error) {
	method := strings.ToUpper(req.Method)
	path := req.URL.Path
	switch {
	case method == http.MethodGet && path == "/v1/subUsers":
		return t.handleListSubUsers(req)
	case method == http.MethodGet && strings.HasPrefix(path, "/v1/regions/") && strings.HasSuffix(path, "/user:describeUserPin"):
		return t.handleDescribeUserPin(req)
	case method == http.MethodPost && path == "/v1/subUser":
		return t.handleCreateSubUser(req, body)
	case method == http.MethodDelete && strings.HasPrefix(path, "/v1/subUser/") && strings.HasSuffix(path, ":detachSubUserPolicy"):
		return t.handleDetachPolicy(req)
	case method == http.MethodPost && strings.HasPrefix(path, "/v1/subUser/") && strings.HasSuffix(path, ":attachSubUserPolicy"):
		return t.handleAttachPolicy(req, body)
	case method == http.MethodDelete && strings.HasPrefix(path, "/v1/subUser/") &&
		!strings.Contains(strings.TrimPrefix(path, "/v1/subUser/"), ":"):
		return t.handleDeleteSubUser(req, path)
	}
	return apiErrorResponse(req, http.StatusNotFound, "NotFound",
		fmt.Sprintf("unsupported iam path: %s %s", method, path)), nil
}

func (t *transport) handleListSubUsers(req *http.Request) (*http.Response, error) {
	resp := api.DescribeSubUsersResponse{RequestID: "req-replay-iam-list"}
	users := t.iam.snapshotUsers()
	resp.Result.Total = len(users)
	for _, user := range users {
		resp.Result.SubUsers = append(resp.Result.SubUsers, api.SubUser{
			Pin:        user.Pin,
			Name:       user.Name,
			Account:    user.Account,
			CreateTime: user.CreateTime,
		})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleDescribeUserPin(req *http.Request) (*http.Response, error) {
	ak := strings.TrimSpace(req.URL.Query().Get("accessKey"))
	if ak != demoCredentials.AccessKey {
		return apiErrorResponse(req, http.StatusForbidden, "AccessKeyNotFound",
			fmt.Sprintf("access key %s not found", ak)), nil
	}
	resp := api.DescribeUserPinResponse{RequestID: "req-replay-user-pin"}
	resp.Result.Pin = demoMasterPin
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleCreateSubUser(req *http.Request, body []byte) (*http.Response, error) {
	var payload api.CreateSubUserRequest
	_ = json.Unmarshal(body, &payload)
	name := strings.TrimSpace(payload.CreateSubUserInfo.Name)
	if name == "" {
		return apiErrorResponse(req, http.StatusBadRequest, "InvalidParameter",
			"sub user name is required"), nil
	}
	user := t.iam.ensureUser(name)
	resp := api.CreateSubUserResponse{RequestID: "req-replay-iam-create"}
	resp.Result.SubUser = api.CreateSubUserResult{
		Pin:     user.Pin,
		Name:    user.Name,
		Account: user.Account,
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleAttachPolicy(req *http.Request, body []byte) (*http.Response, error) {
	rest := strings.TrimPrefix(req.URL.Path, "/v1/subUser/")
	user := strings.TrimSuffix(rest, ":attachSubUserPolicy")
	var payload api.AttachSubUserPolicyRequest
	_ = json.Unmarshal(body, &payload)
	t.iam.attachPolicy(user, payload.PolicyName)
	resp := api.AttachSubUserPolicyResponse{RequestID: "req-replay-iam-attach"}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleDetachPolicy(req *http.Request) (*http.Response, error) {
	rest := strings.TrimPrefix(req.URL.Path, "/v1/subUser/")
	user := strings.TrimSuffix(rest, ":detachSubUserPolicy")
	policy := strings.TrimSpace(req.URL.Query().Get("policyName"))
	if !t.iam.detachPolicy(user, policy) {
		// idempotent: still success
	}
	resp := api.DetachSubUserPolicyResponse{RequestID: "req-replay-iam-detach"}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleDeleteSubUser(req *http.Request, path string) (*http.Response, error) {
	user := strings.TrimPrefix(path, "/v1/subUser/")
	if !t.iam.deleteUser(user) {
		return apiErrorResponse(req, http.StatusNotFound, "ResourceNotFound",
			fmt.Sprintf("sub user %s not found", user)), nil
	}
	resp := api.DeleteSubUserResponse{RequestID: "req-replay-iam-delete"}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}
