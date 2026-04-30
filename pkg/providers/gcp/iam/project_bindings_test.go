package iam

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func TestProjectBindingsRoundTripPreservesEtag(t *testing.T) {
	var setBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/v1/projects/proj-1:getIamPolicy":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST; got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":3,"etag":"etag-7","bindings":[{"role":"roles/viewer","members":["user:a@x"]}]}`))
		case "/v1/projects/proj-1:setIamPolicy":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			setBody = body
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":3,"etag":"etag-8","bindings":[{"role":"roles/viewer","members":["user:a@x","user:b@x"]}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Projects: []string{"proj-1"}, Client: newRMClient(t, server)}
	updated, err := driver.AddBinding(context.Background(), "proj-1", "roles/viewer", "user:b@x")
	if err != nil {
		t.Fatalf("AddBinding: %v", err)
	}
	if updated.Etag != "etag-8" {
		t.Errorf("expected new etag etag-8; got %s", updated.Etag)
	}
	var sent api.SetIamPolicyRequest
	if err := json.Unmarshal(setBody, &sent); err != nil {
		t.Fatalf("unmarshal set body: %v", err)
	}
	if sent.Policy.Etag != "etag-7" {
		t.Errorf("expected GET etag etag-7 to be round-tripped; got %s", sent.Policy.Etag)
	}
	if len(sent.Policy.Bindings) != 1 {
		t.Fatalf("unexpected binding count: %d", len(sent.Policy.Bindings))
	}
	if len(sent.Policy.Bindings[0].Members) != 2 {
		t.Fatalf("expected 2 members; got %v", sent.Policy.Bindings[0].Members)
	}
}

func TestRemoveBindingPrunesEmptyBinding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/v1/projects/proj-1:getIamPolicy":
			_, _ = w.Write([]byte(`{"etag":"e","bindings":[{"role":"roles/viewer","members":["user:a@x"]}]}`))
		case "/v1/projects/proj-1:setIamPolicy":
			body, _ := io.ReadAll(r.Body)
			var sent api.SetIamPolicyRequest
			if err := json.Unmarshal(body, &sent); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(sent.Policy.Bindings) != 0 {
				t.Errorf("expected pruned bindings; got %+v", sent.Policy.Bindings)
			}
			_, _ = w.Write([]byte(`{"etag":"e2","bindings":[]}`))
		}
	}))
	defer server.Close()

	driver := &Driver{Projects: []string{"proj-1"}, Client: newRMClient(t, server)}
	if _, err := driver.RemoveBinding(context.Background(), "proj-1", "roles/viewer", "user:a@x"); err != nil {
		t.Fatalf("RemoveBinding: %v", err)
	}
}

func TestMutatePolicyDoesNotDuplicateMember(t *testing.T) {
	policy := api.IamPolicy{
		Etag:     "e",
		Bindings: []api.Binding{{Role: "roles/viewer", Members: []string{"user:a@x"}}},
	}
	updated := mutatePolicy(policy, "roles/viewer", "user:a@x", true)
	if len(updated.Bindings[0].Members) != 1 {
		t.Errorf("expected single member; got %v", updated.Bindings[0].Members)
	}
}

func newRMClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	httpClient := server.Client()
	transport, err := testutil.RewriteHostsTransport(httpClient.Transport, server.URL,
		"cloudresourcemanager.googleapis.com",
		"iam.googleapis.com",
	)
	if err != nil {
		t.Fatalf("RewriteHostsTransport: %v", err)
	}
	httpClient.Transport = transport
	ts := auth.NewTokenSource(auth.Credential{
		Type:          "service_account",
		ProjectID:     "proj-1",
		PrivateKeyID:  "kid-1",
		PrivateKeyPEM: testutil.PKCS8PrivateKeyPEM,
		ClientEmail:   "demo@example.com",
		TokenURI:      server.URL + "/token",
		Scopes:        []string{auth.DefaultScope},
	}, httpClient)
	return api.NewClient(ts, api.WithHTTPClient(httpClient))
}
