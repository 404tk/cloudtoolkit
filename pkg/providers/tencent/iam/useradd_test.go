package iam

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddUserCreatesUserAttachesPolicyAndPrintsLoginURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "AddUser":
			if body := readBody(t, r); body != `{"Name":"alice","ConsoleLogin":1,"Password":"Secret!1","NeedResetPassword":0}` {
				t.Fatalf("unexpected AddUser body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"Uin":1001,"Name":"alice","RequestId":"req-add-user"}}`))
		case "GetUser":
			if body := readBody(t, r); body != `{"Name":"alice"}` {
				t.Fatalf("unexpected GetUser body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"Uin":1001,"Name":"alice","RequestId":"req-get-user"}}`))
		case "AttachUserPolicy":
			if body := readBody(t, r); body != `{"PolicyId":1,"AttachUin":1001}` {
				t.Fatalf("unexpected AttachUserPolicy body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-attach-user-policy"}}`))
		case "GetUserAppId":
			if body := readBody(t, r); body != "{}" {
				t.Fatalf("unexpected GetUserAppId body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"OwnerUin":"1234567890","RequestId":"req-get-appid"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.UserName = "alice"
	driver.Password = "Secret!1"

	output := captureStdout(t, func() {
		driver.AddUser()
	})

	if !strings.Contains(output, "alice") {
		t.Fatalf("expected username in output: %s", output)
	}
	if !strings.Contains(output, "Secret!1") {
		t.Fatalf("expected password in output: %s", output)
	}
	if !strings.Contains(output, "https://cloud.tencent.com/login/subAccount/1234567890") {
		t.Fatalf("expected login URL in output: %s", output)
	}
}
