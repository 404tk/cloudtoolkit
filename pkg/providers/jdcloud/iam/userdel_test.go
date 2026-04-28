package iam

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDriverDelUserDetachThenDelete(t *testing.T) {
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/subUser/demo-user:detachSubUserPolicy":
			actions = append(actions, "DetachSubUserPolicy")
			if r.URL.RawPath != "/v1/subUser/demo-user%3AdetachSubUserPolicy" {
				t.Fatalf("expected wire RawPath to escape ':' as %%3A, got %q", r.URL.RawPath)
			}
			if got := r.URL.Query().Get("policyName"); got != administratorPolicyName {
				t.Fatalf("expected policyName=%s in query, got %q", administratorPolicyName, got)
			}
			_, _ = w.Write([]byte(`{"requestId":"req-detach","result":{}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/subUser/demo-user":
			actions = append(actions, "DeleteSubUser")
			_, _ = w.Write([]byte(`{"requestId":"req-delete","result":{}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:   newTestClient(server.URL),
		UserName: "demo-user",
	}
	result, err := driver.DelUser()

	if err != nil {
		t.Fatalf("DelUser failed: %v", err)
	}
	if got := strings.Join(actions, ","); got != "DetachSubUserPolicy,DeleteSubUser" {
		t.Fatalf("unexpected actions: %s", got)
	}
	if result.Username != "demo-user" {
		t.Fatalf("unexpected username: %s", result.Username)
	}
	if !strings.Contains(result.Message, "deleted") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestDriverDelUserIgnoresMissingAttachment(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "404 english",
			status: http.StatusNotFound,
			body:   `{"requestId":"req-detach","error":{"status":"NOT_FOUND","code":404,"message":"policy attachment does not exist"}}`,
		},
		{
			// JDCloud's real-world response: 200 HTTP, business code 1011, Chinese message.
			name:   "200 code 1011 chinese",
			status: http.StatusOK,
			body:   `{"requestId":"req-detach","error":{"status":"ERROR","code":1011,"message":"[IAM] 资源不存在,策略不存在"}}`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var actions []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/subUser/demo-user:detachSubUserPolicy":
					actions = append(actions, "DetachSubUserPolicy")
					if tc.status != http.StatusOK {
						w.WriteHeader(tc.status)
					}
					_, _ = w.Write([]byte(tc.body))
				case "/v1/subUser/demo-user":
					actions = append(actions, "DeleteSubUser")
					_, _ = w.Write([]byte(`{"requestId":"req-delete","result":{}}`))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			defer server.Close()

			driver := &Driver{
				Client:   newTestClient(server.URL),
				UserName: "demo-user",
			}
			driver.DelUser()

			if got := strings.Join(actions, ","); got != "DetachSubUserPolicy,DeleteSubUser" {
				t.Fatalf("unexpected actions: %s", got)
			}
		})
	}
}

func TestDriverDelUserStopsOnHardDetachFailure(t *testing.T) {
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/subUser/demo-user:detachSubUserPolicy" {
			t.Fatalf("delete must not be reached on hard detach failure, got %s", r.URL.Path)
		}
		actions = append(actions, "DetachSubUserPolicy")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"requestId":"req-fail","error":{"status":"INTERNAL","code":500,"message":"upstream error"}}`))
	}))
	defer server.Close()

	driver := &Driver{
		Client:   newTestClient(server.URL),
		UserName: "demo-user",
	}
	driver.DelUser()

	if got := strings.Join(actions, ","); got != "DetachSubUserPolicy" {
		t.Fatalf("unexpected actions: %s", got)
	}
}
