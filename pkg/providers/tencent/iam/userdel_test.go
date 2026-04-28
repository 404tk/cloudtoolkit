package iam

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestDelUserDetachesPolicyThenDeletesUser(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "GetUser":
			if body := readBody(t, r); body != `{"Name":"alice"}` {
				t.Fatalf("unexpected GetUser body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"Uin":1001,"Name":"alice","RequestId":"req-get-user"}}`))
		case "DetachUserPolicy":
			if body := readBody(t, r); body != `{"PolicyId":1,"DetachUin":1001}` {
				t.Fatalf("unexpected DetachUserPolicy body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-detach-user-policy"}}`))
		case "DeleteUser":
			if body := readBody(t, r); body != `{"Name":"alice","Force":1}` {
				t.Fatalf("unexpected DeleteUser body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-delete-user"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.UserName = "alice"
	result, err := driver.DelUser()

	if err != nil {
		t.Fatalf("DelUser failed: %v", err)
	}
	if result.Username != "alice" {
		t.Fatalf("unexpected username: %s", result.Username)
	}
	if !strings.Contains(result.Message, "deleted") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}
