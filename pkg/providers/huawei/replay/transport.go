package replay

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type transport struct {
	iam       *iamMutationState
	mu        sync.Mutex
	bucketACL map[string]string
}

func newTransport() *transport {
	return &transport{
		iam:       newIAMMutationState(),
		bucketACL: seedHuaweiBucketACL(),
	}
}

// seedHuaweiBucketACL gives every demo OBS bucket a starting "private" canned
// ACL so audit/expose/audit/unexpose cycles surface deterministic state.
func seedHuaweiBucketACL() map[string]string {
	out := make(map[string]string, len(demoOBSBuckets))
	for _, bucket := range demoOBSBuckets {
		out[bucket.Name] = "private"
	}
	return out
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := demoreplay.ReadRequestBody(req)
	if err != nil {
		return nil, err
	}
	host := normalizeHost(req.URL.Hostname())
	service, region := classifyHost(host)

	switch service {
	case "obs":
		return t.handleOBS(req, host, region, body)
	case "":
		return apiErrorResponse(req, http.StatusNotFound, "InvalidEndpoint",
			fmt.Sprintf("unsupported replay host: %s", host)), nil
	}

	switch verifyOpenAPIAuth(req, body) {
	case demoreplay.AuthInvalidAccessKey:
		return apiErrorResponse(req, http.StatusUnauthorized, "APIGW.0301",
			"Incorrect IAM authentication information: AccessKey not found."), nil
	case demoreplay.AuthInvalidSignature:
		return apiErrorResponse(req, http.StatusUnauthorized, "APIGW.0301",
			"Incorrect IAM authentication information: verify aksk signature fail."), nil
	}

	switch service {
	case "iam":
		return t.handleIAM(req, region, body)
	case "ecs":
		return t.handleECS(req, region, body)
	case "rds":
		return t.handleRDS(req, region, body)
	case "bss":
		return t.handleBSS(req, body)
	case "cts":
		return t.handleCTS(req, region)
	}
	return apiErrorResponse(req, http.StatusNotFound, "InvalidEndpoint",
		fmt.Sprintf("unsupported replay service: %s", service)), nil
}

func classifyHost(host string) (string, string) {
	host = normalizeHost(host)
	if host == "" {
		return "", ""
	}
	switch {
	case host == "bss.myhuaweicloud.com" || host == "bss-intl.myhuaweicloud.com":
		return "bss", ""
	case strings.HasPrefix(host, "obs."):
		return "obs", trimSuffix(strings.TrimPrefix(host, "obs."), ".myhuaweicloud.com")
	case strings.HasPrefix(host, "iam."):
		return "iam", trimSuffix(strings.TrimPrefix(host, "iam."), ".myhuaweicloud.com")
	case strings.HasPrefix(host, "ecs."):
		return "ecs", trimSuffix(strings.TrimPrefix(host, "ecs."), ".myhuaweicloud.com")
	case strings.HasPrefix(host, "rds."):
		return "rds", trimSuffix(strings.TrimPrefix(host, "rds."), ".myhuaweicloud.com")
	case strings.HasPrefix(host, "cts."):
		return "cts", trimSuffix(strings.TrimPrefix(host, "cts."), ".myhuaweicloud.com")
	}
	return "", ""
}

func trimSuffix(value, suffix string) string {
	value = strings.TrimSuffix(value, suffix)
	return strings.TrimSuffix(value, ".")
}

func verifyOpenAPIAuth(req *http.Request, body []byte) demoreplay.AuthFailureKind {
	authHeader := strings.TrimSpace(req.Header.Get(api.HeaderAuthorization))
	parsed, ok := parseSDKAuth(authHeader)
	if !ok {
		return demoreplay.AuthInvalidSignature
	}
	if parsed.AccessKey != demoCredentials.AccessKey {
		return demoreplay.AuthInvalidAccessKey
	}
	xDate := strings.TrimSpace(req.Header.Get(api.HeaderXDate))
	timestamp, err := time.Parse(api.BasicDateFormat, xDate)
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	host := normalizeHost(req.Host)
	if host == "" {
		host = normalizeHost(req.URL.Host)
	}
	headers := flattenSignedHeaders(req.Header)
	signed, err := api.Sign(&api.SignRequest{
		Method:    req.Method,
		Host:      host,
		Path:      req.URL.Path,
		Query:     httpclient.CloneValues(req.URL.Query()),
		Headers:   headers,
		Body:      body,
		AccessKey: demoCredentials.AccessKey,
		SecretKey: demoCredentials.SecretKey,
		Timestamp: timestamp,
	})
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	if demoreplay.SubtleEqual(strings.TrimSpace(signed[api.HeaderAuthorization]), authHeader) {
		return demoreplay.AuthOK
	}
	return demoreplay.AuthInvalidSignature
}

type sdkAuth struct {
	AccessKey     string
	SignedHeaders []string
}

func parseSDKAuth(value string) (sdkAuth, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return sdkAuth{}, false
	}
	if !strings.HasPrefix(value, api.Algorithm+" ") {
		return sdkAuth{}, false
	}
	rest := strings.TrimPrefix(value, api.Algorithm+" ")
	parts := strings.Split(rest, ",")
	parsed := sdkAuth{}
	for _, part := range parts {
		entry := strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(entry, "Access="):
			parsed.AccessKey = strings.TrimPrefix(entry, "Access=")
		case strings.HasPrefix(entry, "SignedHeaders="):
			signed := strings.TrimPrefix(entry, "SignedHeaders=")
			parsed.SignedHeaders = strings.Split(signed, ";")
		}
	}
	if parsed.AccessKey == "" {
		return sdkAuth{}, false
	}
	return parsed, true
}

func flattenSignedHeaders(headers http.Header) map[string]string {
	flattened := make(map[string]string, len(headers))
	for key, values := range headers {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		switch strings.ToLower(name) {
		case strings.ToLower(api.HeaderAuthorization),
			strings.ToLower(api.HeaderXDate),
			strings.ToLower(api.HeaderContentSha256),
			"host":
			continue
		}
		flattened[name] = strings.Join(values, ",")
	}
	return flattened
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
