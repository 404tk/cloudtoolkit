package oss

import (
	"net/http"
	"net/url"
	"testing"
)

func TestBuildStringToSignUsesOSSHeadersAndOnlySignableQueryKeys(t *testing.T) {
	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme:   "https",
			Host:     "examplebucket.oss-cn-shanghai.aliyuncs.com",
			Path:     "/",
			RawQuery: "list-type=2&encoding-type=url&max-keys=100&continuation-token=page-2",
		},
		Header: make(http.Header),
	}
	req.Header.Set("Date", "Sun, 19 Apr 2026 12:00:00 GMT")
	req.Header.Set("X-Oss-Security-Token", "sts-token")

	got := buildStringToSign(req, "examplebucket")
	want := "GET\n\n\nSun, 19 Apr 2026 12:00:00 GMT\nx-oss-security-token:sts-token\n/examplebucket/?continuation-token=page-2"
	if got != want {
		t.Fatalf("unexpected string to sign:\n got: %q\nwant: %q", got, want)
	}
}
