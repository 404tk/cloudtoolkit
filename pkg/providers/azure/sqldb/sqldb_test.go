package sqldb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
)

func newDriverClient(server *httptest.Server) *azapi.Client {
	cred := auth.New("client", "secret", "tenant", "sub", "")
	ts := auth.NewTokenSource(cred, server.Client())
	auth.SetCachedToken(ts, auth.Token{AccessToken: "demo", ExpiresAt: time.Now().Add(time.Hour)})
	endpoints := cloud.For(cred.Cloud)
	return azapi.NewClient(ts, endpoints, azapi.WithBaseURL(server.URL), azapi.WithHTTPClient(server.Client()))
}

func TestCreateAccountSendsPasswordRotation(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkadmin:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/Microsoft.Sql/servers/sql-1") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body azapi.SQLServerPatch
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Properties.AdministratorLoginPassword != "Ctk!Pwd2026" {
			t.Fatalf("unexpected password: %s", body.Properties.AdministratorLoginPassword)
		}
		_, _ = w.Write([]byte(`{"id":"sub","name":"sql-1","location":"eastus","properties":{"state":"Ready"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(server), SubscriptionIDs: []string{"sub"}}
	res, err := driver.CreateAccount(context.Background(), "rg-1/sql-1")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if res.Username != "ctkadmin" || res.Password != "Ctk!Pwd2026" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestDeleteAccountSendsRandomPassword(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkadmin:Ctk!Pwd2026"})
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body azapi.SQLServerPatch
		_ = json.NewDecoder(r.Body).Decode(&body)
		captured = body.Properties.AdministratorLoginPassword
		_, _ = w.Write([]byte(`{"id":"sub","name":"sql-1","location":"eastus","properties":{"state":"Ready"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(server), SubscriptionIDs: []string{"sub"}}
	if _, err := driver.DeleteAccount(context.Background(), "rg-1/sql-1"); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if captured == "" || captured == "Ctk!Pwd2026" {
		t.Errorf("expected random password, got %q", captured)
	}
}

func TestCreateAccountRejectsBadInstanceID(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkadmin:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("API should not be called")
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(server), SubscriptionIDs: []string{"sub"}}
	if _, err := driver.CreateAccount(context.Background(), "no-slash"); err == nil {
		t.Fatalf("expected error for malformed instance id")
	}
}

func TestCreateAccountRejectsMissingConfig(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("API should not be called")
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(server), SubscriptionIDs: []string{"sub"}}
	if _, err := driver.CreateAccount(context.Background(), "rg-1/sql-1"); err == nil {
		t.Fatalf("expected error for missing rds-account-check")
	}
}
