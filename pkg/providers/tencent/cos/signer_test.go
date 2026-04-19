package cos

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

func TestBuildAuthorizationMatchesSDKReference(t *testing.T) {
	const expectAuthorization = "q-sign-algorithm=sha1&q-ak=QmFzZTY0IGlzIGEgZ2VuZXJp&q-sign-time=1480932292;1481012292&q-key-time=1480932292;1481012292&q-header-list=host;x-cos-content-sha1;x-cos-stroage-class&q-url-param-list=&q-signature=ce4ac0ecbcdb30538b3fee0a97cc6389694ce53a"

	req, err := http.NewRequest(http.MethodPut, "http://testbucket-125000000.cos.ap-guangzhou.myqcloud.com/testfile2", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Add("Host", "testbucket-125000000.cos.ap-guangzhou.myqcloud.com")
	req.Header.Add("x-cos-content-sha1", "db8ac1c259eb89d4a131b253bacfca5f319d54f2")
	req.Header.Add("x-cos-stroage-class", "nearline")

	window := &authTime{
		SignStartTime: time.Unix(1480932292, 0),
		SignEndTime:   time.Unix(1481012292, 0),
		KeyStartTime:  time.Unix(1480932292, 0),
		KeyEndTime:    time.Unix(1481012292, 0),
	}
	got, err := buildAuthorization("QmFzZTY0IGlzIGEgZ2VuZXJp", "AKIDZfbOA78asKUYBcXFrJD0a1ICvR98JM", req, window, true)
	if err != nil {
		t.Fatalf("buildAuthorization() error = %v", err)
	}
	if got != expectAuthorization {
		t.Fatalf("unexpected authorization:\n got:  %s\nwant: %s", got, expectAuthorization)
	}
}

func TestSignAddsSecurityToken(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://service.cos.myqcloud.com/", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	cred := auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "TOKENEXAMPLE")
	if err := Sign(req, cred, time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if req.Header.Get("x-cos-security-token") != "TOKENEXAMPLE" {
		t.Fatalf("unexpected token header: %q", req.Header.Get("x-cos-security-token"))
	}
	if got := req.Header.Get("Host"); got != "service.cos.myqcloud.com" {
		t.Fatalf("unexpected host header: %q", got)
	}
	if got := req.Header.Get("Authorization"); got == "" || !containsAll(got, "q-header-list=host;x-cos-security-token", "q-sign-algorithm=sha1", "q-ak=AKIDEXAMPLE") {
		t.Fatalf("unexpected authorization header: %q", got)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
