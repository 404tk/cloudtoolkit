package sqladmin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
)

func newDriverClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	httpClient := server.Client()
	transport, err := testutil.RewriteHostsTransport(httpClient.Transport, server.URL, "sqladmin.googleapis.com")
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

func TestCreateAccountSendsExpectedPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/instances/sql-1/users") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"op-1","status":"DONE","operationType":"CREATE_USER"}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(t, server), Projects: []string{"proj-1"}}
	res, err := driver.CreateAccount(context.Background(), "sql-1")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if res.Username != "ctkuser" || res.Password != "Ctk!Pwd2026" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestDeleteAccountSendsExpectedPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.URL.Query().Get("name"); got != "ctkuser" {
			t.Fatalf("unexpected name param: %s", got)
		}
		_, _ = w.Write([]byte(`{"name":"op-2","status":"DONE","operationType":"DELETE_USER"}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(t, server), Projects: []string{"proj-1"}}
	res, err := driver.DeleteAccount(context.Background(), "sql-1")
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if res.Username != "ctkuser" {
		t.Errorf("unexpected username: %s", res.Username)
	}
}

func TestCreateAccountRejectsMissingConfig(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("API should not be called")
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(t, server), Projects: []string{"proj-1"}}
	if _, err := driver.CreateAccount(context.Background(), "sql-1"); err == nil {
		t.Fatalf("expected error for missing rds-account-check")
	}
}

func TestCreateAccountRejectsNoProject(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("API should not be called")
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(t, server), Projects: nil}
	if _, err := driver.CreateAccount(context.Background(), "sql-1"); err == nil {
		t.Fatalf("expected error for missing project")
	}
}
