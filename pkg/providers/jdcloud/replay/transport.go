package replay

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type transport struct {
	mu        sync.Mutex
	iam       *iamMutationState
	bucketACL map[string]string
}

func newTransport() *transport {
	return &transport{
		iam:       newIAMState(),
		bucketACL: seedBucketACL(),
	}
}

func seedBucketACL() map[string]string {
	out := make(map[string]string, len(demoBuckets))
	for _, bucket := range demoBuckets {
		out[bucket.Name] = "private"
	}
	return out
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := demoreplay.ReadRequestBody(req)
	if err != nil {
		return nil, err
	}
	host := normalizeHost(firstNonEmpty(req.Host, req.URL.Hostname()))
	if isOSSDataPlaneHost(host) {
		return t.handleOSSDataPlane(req, host, body)
	}
	service := serviceFromHost(host)
	if service == "" {
		return apiErrorResponse(req, http.StatusNotFound, "InvalidEndpoint",
			fmt.Sprintf("unsupported replay host: %s", host)), nil
	}

	switch verifyAuth(req, body, service) {
	case demoreplay.AuthInvalidAccessKey:
		return apiErrorResponse(req, http.StatusUnauthorized, "InvalidAccessKeyId",
			"The Access Key Id you provided does not exist."), nil
	case demoreplay.AuthInvalidSignature:
		return apiErrorResponse(req, http.StatusUnauthorized, "SignatureDoesNotMatch",
			"The request signature does not match."), nil
	}

	switch service {
	case "iam":
		return t.handleIAM(req, body)
	case "vm":
		return t.handleVM(req)
	case "lavm":
		return t.handleLAVM(req)
	case "oss":
		return t.handleOSS(req)
	case "asset":
		return t.handleAsset(req)
	}
	return apiErrorResponse(req, http.StatusNotFound, "InvalidService",
		fmt.Sprintf("unsupported replay service: %s", service)), nil
}

func serviceFromHost(host string) string {
	host = normalizeHost(host)
	suffix := ".jdcloud-api.com"
	if !strings.HasSuffix(host, suffix) {
		return ""
	}
	return strings.TrimSuffix(host, suffix)
}

func isOSSDataPlaneHost(host string) bool {
	host = normalizeHost(host)
	return strings.HasPrefix(host, "s3.") && strings.HasSuffix(host, ".jdcloud-oss.com")
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if v := strings.TrimSpace(value); v != "" {
			return v
		}
	}
	return ""
}

func verifyAuth(req *http.Request, body []byte, service string) demoreplay.AuthFailureKind {
	header := strings.TrimSpace(req.Header.Get(api.HeaderAuthorization))
	parsed, ok := parseAuth(header)
	if !ok {
		return demoreplay.AuthInvalidSignature
	}
	if parsed.AccessKey != demoCredentials.AccessKey {
		return demoreplay.AuthInvalidAccessKey
	}
	xDate := strings.TrimSpace(req.Header.Get(api.HeaderXJdcloudDate))
	timestamp, err := time.Parse(api.TimeFormat, xDate)
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	nonce := strings.TrimSpace(req.Header.Get(api.HeaderXJdcloudNonce))
	if nonce == "" {
		return demoreplay.AuthInvalidSignature
	}
	host := normalizeHost(firstNonEmpty(req.Host, req.URL.Host))
	contentType := strings.TrimSpace(req.Header.Get("Content-Type"))
	signed, err := api.Sign(api.SignInput{
		Method:       req.Method,
		Host:         host,
		Path:         req.URL.Path,
		Query:        httpclient.CloneValues(req.URL.Query()),
		Body:         body,
		ContentType:  contentType,
		Service:      service,
		Region:       parsed.Region,
		AccessKey:    demoCredentials.AccessKey,
		SecretKey:    demoCredentials.SecretKey,
		SessionToken: strings.TrimSpace(req.Header.Get(api.HeaderXJdcloudToken)),
		Nonce:        nonce,
		Timestamp:    timestamp,
		Headers:      req.Header.Clone(),
	})
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	if demoreplay.SubtleEqual(strings.TrimSpace(signed.Authorization), header) {
		return demoreplay.AuthOK
	}
	return demoreplay.AuthInvalidSignature
}

type parsedAuth struct {
	AccessKey string
	Region    string
	Service   string
}

func parseAuth(value string) (parsedAuth, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, api.Algorithm+" ") {
		return parsedAuth{}, false
	}
	rest := strings.TrimPrefix(value, api.Algorithm+" ")
	parts := strings.Split(rest, ",")
	parsed := parsedAuth{}
	for _, part := range parts {
		entry := strings.TrimSpace(part)
		if !strings.HasPrefix(entry, "Credential=") {
			continue
		}
		credential := strings.TrimPrefix(entry, "Credential=")
		scope := strings.Split(credential, "/")
		if len(scope) < 5 {
			return parsedAuth{}, false
		}
		parsed.AccessKey = strings.TrimSpace(scope[0])
		parsed.Region = strings.TrimSpace(scope[2])
		parsed.Service = strings.TrimSpace(scope[3])
	}
	if parsed.AccessKey == "" {
		return parsedAuth{}, false
	}
	return parsed, true
}

type errorEnvelope struct {
	RequestID string            `json:"requestId"`
	Error     *api.APIErrorBody `json:"error,omitempty"`
	Result    map[string]any    `json:"result"`
}

func apiErrorResponse(req *http.Request, statusCode int, status, message string) *http.Response {
	envelope := errorEnvelope{
		RequestID: "req-replay-jdcloud",
		Error: &api.APIErrorBody{
			Code:    statusCode,
			Status:  strings.TrimSpace(status),
			Message: strings.TrimSpace(message),
		},
		Result: map[string]any{},
	}
	resp := demoreplay.JSONResponse(req, statusCode, envelope)
	resp.Header.Set("X-Jdcloud-Request-Id", envelope.RequestID)
	return resp
}
