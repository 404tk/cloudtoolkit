package iam

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
)

func TestDriverAddUserCreatesSubUserAndAttachesAdminPolicy(t *testing.T) {
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/subUser":
			actions = append(actions, "CreateSubUser")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			var req api.CreateSubUserRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			if req.CreateSubUserInfo.Name != "demo-user" {
				t.Fatalf("unexpected name: %q", req.CreateSubUserInfo.Name)
			}
			if req.CreateSubUserInfo.Password != "TempPassw0rd!" {
				t.Fatalf("unexpected password: %q", req.CreateSubUserInfo.Password)
			}
			if req.CreateSubUserInfo.ConsoleLogin == nil || !*req.CreateSubUserInfo.ConsoleLogin {
				t.Fatalf("expected consoleLogin=true, got %+v", req.CreateSubUserInfo.ConsoleLogin)
			}
			if req.CreateSubUserInfo.CreateAk == nil || *req.CreateSubUserInfo.CreateAk {
				t.Fatalf("expected createAk=false, got %+v", req.CreateSubUserInfo.CreateAk)
			}
			_, _ = w.Write([]byte(`{"requestId":"req-create","result":{"subUser":{"name":"demo-user","account":"1001"}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/subUser/demo-user:attachSubUserPolicy":
			actions = append(actions, "AttachSubUserPolicy")
			// JDCloud signs with the colon percent-encoded; asserting RawPath
			// here pins the HTTP wire form to match what the signer hashes.
			if r.URL.RawPath != "/v1/subUser/demo-user%3AattachSubUserPolicy" {
				t.Fatalf("expected wire RawPath to escape ':' as %%3A, got %q", r.URL.RawPath)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			var req api.AttachSubUserPolicyRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode attach body: %v", err)
			}
			if req.SubUser != "demo-user" || req.PolicyName != administratorPolicyName {
				t.Fatalf("unexpected attach payload: %+v", req)
			}
			_, _ = w.Write([]byte(`{"requestId":"req-attach","result":{}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/regions/cn-north-1/user:describeUserPin":
			actions = append(actions, "DescribeUserPin")
			if got := r.URL.Query().Get("accessKey"); got != "AKID" {
				t.Fatalf("expected accessKey=AKID in query, got %q", got)
			}
			_, _ = w.Write([]byte(`{"requestId":"req-pin","result":{"pin":"jd_master_demo"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:    newTestClient(server.URL),
		AccessKey: "AKID",
		UserName:  "demo-user",
		Password:  "TempPassw0rd!",
	}
	driver.AddUser()

	if got := strings.Join(actions, ","); got != "CreateSubUser,AttachSubUserPolicy,DescribeUserPin" {
		t.Fatalf("unexpected actions: %s", got)
	}
}

func TestDriverAddUserIgnoresPinLookupFailure(t *testing.T) {
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/subUser":
			actions = append(actions, "CreateSubUser")
			_, _ = w.Write([]byte(`{"requestId":"req-create","result":{"subUser":{"name":"demo-user","account":"1001"}}}`))
		case r.URL.Path == "/v1/subUser/demo-user:attachSubUserPolicy":
			actions = append(actions, "AttachSubUserPolicy")
			_, _ = w.Write([]byte(`{"requestId":"req-attach","result":{}}`))
		case r.URL.Path == "/v1/regions/cn-north-1/user:describeUserPin":
			actions = append(actions, "DescribeUserPin")
			if got := r.URL.Query().Get("accessKey"); got != "AKID" {
				t.Fatalf("expected accessKey=AKID in query, got %q", got)
			}
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"requestId":"req-pin-403","error":{"status":"HTTP_FORBIDDEN","code":403,"message":"gateway"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:    newTestClient(server.URL),
		AccessKey: "AKID",
		UserName:  "demo-user",
		Password:  "TempPassw0rd!",
	}
	// AddUser must not abort when DescribeUserPin fails — the account already
	// exists and the login URL just renders without the master pin prefix.
	driver.AddUser()

	if got := strings.Join(actions, ","); got != "CreateSubUser,AttachSubUserPolicy,DescribeUserPin" {
		t.Fatalf("unexpected actions: %s", got)
	}
}

func TestDriverAddUserSkipsAttachOnCreateFailure(t *testing.T) {
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/subUser" {
			t.Fatalf("attach must not be reached when create fails, got %s", r.URL.Path)
		}
		actions = append(actions, "CreateSubUser")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"requestId":"req-err","error":{"status":"BAD_REQUEST","code":400,"message":"password does not meet policy"}}`))
	}))
	defer server.Close()

	driver := &Driver{
		Client:    newTestClient(server.URL),
		AccessKey: "AKID",
		UserName:  "demo-user",
		Password:  "weak",
	}
	driver.AddUser()

	if got := strings.Join(actions, ","); got != "CreateSubUser" {
		t.Fatalf("unexpected actions: %s", got)
	}
}

func TestDriverAddUserRejectsEmptyInputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("no HTTP request expected for empty inputs: %s", r.URL.Path)
	}))
	defer server.Close()

	cases := []struct {
		name string
		user string
		pwd  string
	}{
		{"empty name", "", "TempPassw0rd!"},
		{"empty password", "demo-user", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := &Driver{
				Client:    newTestClient(server.URL),
				AccessKey: "AKID",
				UserName:  tc.user,
				Password:  tc.pwd,
			}
			driver.AddUser()
		})
	}
}
