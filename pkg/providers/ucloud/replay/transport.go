package replay

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type transport struct {
	iam *iamMutationState
}

func newTransport() *transport { return &transport{iam: newIAMState()} }

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := demoreplay.ReadRequestBody(req)
	if err != nil {
		return nil, err
	}
	host := normalizeHost(req.URL.Hostname())
	if host != "api.ucloud.cn" {
		return errorResponse(req, http.StatusNotFound, 404,
			fmt.Sprintf("unsupported replay host: %s", host)), nil
	}

	form, err := url.ParseQuery(string(body))
	if err != nil {
		return errorResponse(req, http.StatusBadRequest, 400, err.Error()), nil
	}
	params := flattenForm(form)
	switch verifyAuth(params) {
	case demoreplay.AuthInvalidAccessKey:
		return errorResponse(req, http.StatusForbidden, 170, "Invalid PublicKey"), nil
	case demoreplay.AuthInvalidSignature:
		return errorResponse(req, http.StatusForbidden, 171, "Signature mismatch"), nil
	}

	action := strings.TrimSpace(params["Action"])
	switch action {
	case "GetUserInfo":
		return t.handleGetUserInfo(req)
	case "GetProjectList":
		return t.handleGetProjectList(req)
	case "GetRegion":
		return t.handleGetRegion(req)
	case "GetBalance":
		return t.handleGetBalance(req)
	case "DescribeUHostInstance":
		return t.handleDescribeUHostInstance(req, params)
	case "DescribeBucket":
		return t.handleDescribeBucket(req, params)
	case "DescribeUDBInstance":
		return t.handleDescribeUDBInstance(req, params)
	case "DescribeUDNSZone":
		return t.handleDescribeUDNSZone(req, params)
	case "DescribeUDNSRecord":
		return t.handleDescribeUDNSRecord(req, params)
	case "ListUsers":
		return t.handleListUsers(req)
	case "CreateUser":
		return t.handleCreateUser(req, params)
	case "DeleteUser":
		return t.handleDeleteUser(req, params)
	case "AttachPoliciesToUser":
		return t.handleAttachPolicies(req, params)
	}
	return errorResponse(req, http.StatusNotFound, 1000,
		fmt.Sprintf("unsupported replay action: %s", action)), nil
}

func flattenForm(values url.Values) map[string]string {
	out := make(map[string]string, len(values))
	for key, list := range values {
		if len(list) > 0 {
			out[key] = list[0]
		}
	}
	return out
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil && u.Host != "" {
			host = u.Host
		}
	}
	host = strings.TrimSuffix(host, ":443")
	host = strings.TrimSuffix(host, ":80")
	return strings.ToLower(host)
}

func verifyAuth(params map[string]string) demoreplay.AuthFailureKind {
	publicKey := strings.TrimSpace(params["PublicKey"])
	if publicKey == "" {
		return demoreplay.AuthInvalidSignature
	}
	if publicKey != demoCredentials.AccessKey {
		return demoreplay.AuthInvalidAccessKey
	}
	provided := strings.TrimSpace(params["Signature"])
	if provided == "" {
		return demoreplay.AuthInvalidSignature
	}

	signing := make(map[string]string, len(params))
	for key, value := range params {
		if key == "Signature" {
			continue
		}
		signing[key] = value
	}
	expected := signRequest(signing, demoCredentials.SecretKey)
	if demoreplay.SubtleEqual(expected, provided) {
		return demoreplay.AuthOK
	}
	return demoreplay.AuthInvalidSignature
}

func signRequest(params map[string]string, secretKey string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString(params[key])
	}
	builder.WriteString(secretKey)
	sum := sha1.Sum([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

type baseEnvelope struct {
	Action  string `json:"Action"`
	RetCode int    `json:"RetCode"`
	Message string `json:"Message,omitempty"`
}

func errorResponse(req *http.Request, statusCode, retCode int, message string) *http.Response {
	type body struct {
		baseEnvelope
	}
	payload := body{
		baseEnvelope: baseEnvelope{
			RetCode: retCode,
			Message: strings.TrimSpace(message),
		},
	}
	resp := demoreplay.JSONResponse(req, statusCode, payload)
	resp.Header.Set("X-UCloud-Request-Id", "req-replay-ucloud")
	return resp
}

func successResponse(req *http.Request, payload any) *http.Response {
	resp := demoreplay.JSONResponse(req, http.StatusOK, payload)
	resp.Header.Set("X-UCloud-Request-Id", "req-replay-ucloud")
	return resp
}

// helper to build the BaseResponse part of any response.
func newBase(action string) api.BaseResponse {
	return api.BaseResponse{Action: action, RetCode: 0, Message: ""}
}
