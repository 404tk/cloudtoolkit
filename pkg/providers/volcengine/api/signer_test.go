package api

import (
	"net/url"
	"testing"
)

func TestCanonicalQueryStringReplacesPlusWithPercent20(t *testing.T) {
	values := url.Values{}
	values.Set("Action", "List Users")
	values.Set("Version", "2018-01-01")
	got := canonicalQueryString(values)
	want := "Action=List%20Users&Version=2018-01-01"
	if got != want {
		t.Fatalf("canonicalQueryString() = %q, want %q", got, want)
	}
}

func TestCanonicalURIEncodesRFC3986(t *testing.T) {
	got := canonicalURI("/bucket name/hello+world")
	want := "/bucket%20name/hello%2Bworld"
	if got != want {
		t.Fatalf("canonicalURI() = %q, want %q", got, want)
	}
}

func TestNormalizeHostStripsDefaultPorts(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "billing.volcengineapi.com:443", want: "billing.volcengineapi.com"},
		{input: "http://iam.volcengineapi.com:80", want: "iam.volcengineapi.com"},
		{input: "ecs.cn-beijing.volcengineapi.com:8443", want: "ecs.cn-beijing.volcengineapi.com:8443"},
	}
	for _, tt := range tests {
		if got := normalizeHost(tt.input); got != tt.want {
			t.Fatalf("normalizeHost(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
