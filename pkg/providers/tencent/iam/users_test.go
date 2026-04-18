package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/404tk/cloudtoolkit/utils"
)

func TestListUsersIncludesAttachedPolicies(t *testing.T) {
	oldListPolicies := utils.ListPolicies
	utils.ListPolicies = true
	policy_infos = nil
	t.Cleanup(func() {
		utils.ListPolicies = oldListPolicies
		policy_infos = nil
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "ListUsers":
			if body := readBody(t, r); body != "{}" {
				t.Fatalf("unexpected ListUsers body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"Data":[{"Uin":1001,"Name":"alice","ConsoleLogin":1,"CreateTime":"2024-01-02 03:04:05"},{"Uin":1002,"Name":"bob","ConsoleLogin":0,"CreateTime":"2024-02-03 04:05:06"}],"RequestId":"req-users"}}`))
		case "ListAttachedUserAllPolicies":
			switch body := readBody(t, r); body {
			case `{"TargetUin":1001,"Rp":20,"Page":1,"AttachType":0}`:
				_, _ = w.Write([]byte(`{"Response":{"PolicyList":[{"PolicyId":"1","PolicyName":"QcloudAccessForFoo","StrategyType":"2"},{"PolicyId":"101","PolicyName":"CustomAccess","StrategyType":"1"}],"TotalNum":2,"RequestId":"req-policies-1"}}`))
			case `{"TargetUin":1002,"Rp":20,"Page":1,"AttachType":0}`:
				_, _ = w.Write([]byte(`{"Response":{"PolicyList":[],"TotalNum":0,"RequestId":"req-policies-2"}}`))
			default:
				t.Fatalf("unexpected ListAttachedUserAllPolicies body: %s", body)
			}
		case "GetPolicy":
			if body := readBody(t, r); body != `{"PolicyId":101}` {
				t.Fatalf("unexpected GetPolicy body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"PolicyDocument":"{\"version\":\"2.0\",\"statement\":[{\"effect\":\"allow\"}]}","RequestId":"req-get-policy"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	users, err := driver.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("unexpected user count: %d", len(users))
	}
	if users[0].UserName != "alice" || users[0].UserId != "1001" {
		t.Fatalf("unexpected first user: %+v", users[0])
	}
	if !users[0].EnableLogin {
		t.Fatalf("expected alice console login enabled")
	}
	if users[0].Policies != "QcloudAccessForFoo\nCustomAccess" {
		t.Fatalf("unexpected alice policies: %q", users[0].Policies)
	}
	if users[1].EnableLogin {
		t.Fatalf("expected bob console login disabled")
	}
	if users[1].Policies != "" {
		t.Fatalf("unexpected bob policies: %q", users[1].Policies)
	}
	if got := policy_infos["CustomAccess"]; got == "" {
		t.Fatalf("expected custom policy cache to be populated")
	}
}
