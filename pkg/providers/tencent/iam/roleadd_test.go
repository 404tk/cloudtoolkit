package iam

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestAddRoleCreatesRoleAttachesPolicyAndLogsSwitchURL(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "CreateRole":
			body := readBody(t, r)
			var payload map[string]any
			if err := json.Unmarshal([]byte(body), &payload); err != nil {
				t.Fatalf("decode CreateRole body: %v", err)
			}
			if payload["RoleName"] != "shadow-admin" {
				t.Fatalf("unexpected role name payload: %v", payload)
			}
			if payload["ConsoleLogin"] != float64(1) {
				t.Fatalf("unexpected console login payload: %v", payload)
			}
			if payload["SessionDuration"] != float64(10000) {
				t.Fatalf("unexpected session duration payload: %v", payload)
			}
			policyDocument, _ := payload["PolicyDocument"].(string)
			if !strings.Contains(policyDocument, `qcs::cam::uin/1234567890:root`) {
				t.Fatalf("unexpected policy document: %s", policyDocument)
			}
			_, _ = w.Write([]byte(`{"Response":{"RoleId":"rid-1","RequestId":"req-create-role"}}`))
		case "AttachRolePolicy":
			if body := readBody(t, r); body != `{"PolicyId":1,"AttachRoleName":"shadow-admin"}` {
				t.Fatalf("unexpected AttachRolePolicy body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"RequestId":"req-attach-role-policy"}}`))
		case "GetUserAppId":
			if body := readBody(t, r); body != "{}" {
				t.Fatalf("unexpected GetUserAppId body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"OwnerUin":"22334455","RequestId":"req-get-appid"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.RoleName = "shadow-admin"
	driver.Uin = "1234567890"
	driver.AddRole()

	if got := buffer.String(); !strings.Contains(got, "https://cloud.tencent.com/cam/switchrole?ownerUin=22334455&roleName=shadow-admin") {
		t.Fatalf("unexpected logger output: %s", got)
	}
}
