package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

const testSAEmail = "ctk-demo@proj-1.iam.gserviceaccount.com"

func TestListKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/v1/projects/proj-1/serviceAccounts/" + testSAEmail + "/keys":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET; got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"keys":[{"name":"projects/proj-1/serviceAccounts/` + testSAEmail + `/keys/abc123","keyType":"USER_MANAGED"}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Projects: []string{"proj-1"}, Client: newSAKeyClient(t, server)}
	keys, err := driver.ListKeys(context.Background(), "proj-1", testSAEmail)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key; got %d", len(keys))
	}
	if KeyShortID(keys[0].Name) != "abc123" {
		t.Errorf("unexpected key id: %s", keys[0].Name)
	}
}

func TestCreateKeyReturnsPrivateKeyData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/v1/projects/proj-1/serviceAccounts/" + testSAEmail + "/keys":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST; got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"projects/proj-1/serviceAccounts/` + testSAEmail + `/keys/new123","privateKeyData":"BASE64_DATA","keyType":"USER_MANAGED"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Projects: []string{"proj-1"}, Client: newSAKeyClient(t, server)}
	key, err := driver.CreateKey(context.Background(), "proj-1", testSAEmail)
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if key.PrivateKeyData != "BASE64_DATA" {
		t.Errorf("expected privateKeyData; got %q", key.PrivateKeyData)
	}
}

func TestDeleteKeyByShortID(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		default:
			expected := "/v1/projects/proj-1/serviceAccounts/" + testSAEmail + "/keys/abc123"
			if r.URL.Path != expected {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE; got %s", r.Method)
			}
			called = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	driver := &Driver{Projects: []string{"proj-1"}, Client: newSAKeyClient(t, server)}
	if err := driver.DeleteKey(context.Background(), "proj-1", testSAEmail, "abc123"); err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
	if !called {
		t.Error("expected DELETE to be called")
	}
}

func TestKeyShortID(t *testing.T) {
	cases := map[string]string{
		"abc":                                "abc",
		"projects/p/serviceAccounts/x/keys/abc": "abc",
		"keys/zzz":                           "zzz",
	}
	for in, want := range cases {
		if got := KeyShortID(in); got != want {
			t.Errorf("KeyShortID(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestServiceAccountResourceID(t *testing.T) {
	got := serviceAccountResourceID("proj-1", testSAEmail)
	if !strings.HasSuffix(got, testSAEmail) || !strings.HasPrefix(got, "projects/proj-1/serviceAccounts/") {
		t.Errorf("unexpected resource id: %s", got)
	}
	// already-canonical input should pass through
	in := "projects/p/serviceAccounts/x"
	if got := serviceAccountResourceID("proj-1", in); got != in {
		t.Errorf("expected pass-through; got %s", got)
	}
}

func newSAKeyClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	httpClient := server.Client()
	transport, err := testutil.RewriteHostsTransport(httpClient.Transport, server.URL,
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
