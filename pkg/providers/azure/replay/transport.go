package replay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type transport struct{}

func newTransport() *transport { return &transport{} }

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := demoreplay.ReadRequestBody(req)
	if err != nil {
		return nil, err
	}
	host := normalizeHost(req.URL.Hostname())
	switch {
	case isLoginHost(host):
		return t.handleToken(req, body)
	case isManagementHost(host):
		if !verifyBearerToken(req) {
			return armErrorResponse(req, http.StatusUnauthorized, "InvalidAuthenticationToken",
				"The access token is invalid."), nil
		}
		return t.handleARM(req)
	}
	return armErrorResponse(req, http.StatusNotFound, "InvalidEndpoint",
		fmt.Sprintf("unsupported replay host: %s", host)), nil
}

func isLoginHost(host string) bool {
	host = normalizeHost(host)
	return host == "login.microsoftonline.com" ||
		host == "login.chinacloudapi.cn" ||
		host == "login.microsoftonline.us" ||
		host == "login.microsoftonline.de"
}

func isManagementHost(host string) bool {
	host = normalizeHost(host)
	return host == "management.azure.com" ||
		host == "management.chinacloudapi.cn" ||
		host == "management.usgovcloudapi.net" ||
		host == "management.microsoftazure.de"
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

func (t *transport) handleToken(req *http.Request, body []byte) (*http.Response, error) {
	if !strings.Contains(req.URL.Path, "/oauth2/") || !strings.HasSuffix(req.URL.Path, "/token") {
		return tokenErrorResponse(req, http.StatusNotFound, "invalid_request",
			"unsupported token endpoint path: "+req.URL.Path), nil
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		return tokenErrorResponse(req, http.StatusBadRequest, "invalid_request", err.Error()), nil
	}
	if grant := strings.TrimSpace(form.Get("grant_type")); grant != "client_credentials" {
		return tokenErrorResponse(req, http.StatusBadRequest, "unsupported_grant_type",
			fmt.Sprintf("unsupported grant_type: %s", grant)), nil
	}
	if id := strings.TrimSpace(form.Get("client_id")); id != demoCredentials.AccessKey {
		return tokenErrorResponse(req, http.StatusUnauthorized, "invalid_client",
			"AADSTS700016: Application not found in the directory."), nil
	}
	if secret := strings.TrimSpace(form.Get("client_secret")); secret != demoCredentials.SecretKey {
		return tokenErrorResponse(req, http.StatusUnauthorized, "invalid_client",
			"AADSTS7000215: Invalid client secret provided."), nil
	}
	resp := struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}{
		AccessToken: demoAccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func tokenErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	payload := map[string]string{
		"error":             strings.TrimSpace(code),
		"error_description": strings.TrimSpace(message),
	}
	return demoreplay.JSONResponse(req, statusCode, payload)
}

func verifyBearerToken(req *http.Request) bool {
	header := strings.TrimSpace(req.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	return demoreplay.SubtleEqual(token, demoAccessToken)
}

type armError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type armErrorBody struct {
	Error armError `json:"error"`
}

func armErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	payload := armErrorBody{Error: armError{
		Code:    strings.TrimSpace(code),
		Message: strings.TrimSpace(message),
	}}
	resp := demoreplay.JSONResponse(req, statusCode, payload)
	resp.Header.Set("x-ms-request-id", "req-replay-azure")
	return resp
}

// jsonResponse returns a JSON 200 response and stamps a fake request ID.
func jsonResponse(req *http.Request, payload any) *http.Response {
	body, _ := json.Marshal(payload)
	resp := demoreplay.Response(req, http.StatusOK, "application/json", body)
	resp.Header.Set("x-ms-request-id", "req-replay-azure")
	return resp
}
