package iam

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestDelRoleDetachesPolicyThenDeletesRole(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "DetachRolePolicy":
			if body := readBody(t, r); body != `{"PolicyId":1,"DetachRoleName":"shadow-admin"}` {
				t.Fatalf("unexpected DetachRolePolicy body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-detach-role-policy"}}`))
		case "DeleteRole":
			if body := readBody(t, r); body != `{"RoleName":"shadow-admin"}` {
				t.Fatalf("unexpected DeleteRole body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-delete-role"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.RoleName = "shadow-admin"
	driver.DelRole()

	if got := buffer.String(); !strings.Contains(got, "shadow-admin role delete completed.") {
		t.Fatalf("unexpected logger output: %s", got)
	}
}
