package udb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
)

func newDriver(baseURL string) *Driver {
	credential := auth.New("ucloudpubkey-EXAMPLE", "ucloudprivkey-EXAMPLE", "")
	return &Driver{
		Credential: credential,
		Client: api.NewClient(credential,
			api.WithBaseURL(baseURL),
			api.WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		),
		Regions: []string{"cn-bj2"},
	}
}

func TestCreateAccountSendsExpectedPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.Form.Get("Action"); got != "CreateUDBUser" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := r.Form.Get("UserName"); got != "ctkuser" {
			t.Fatalf("unexpected user: %s", got)
		}
		if got := r.Form.Get("DBId"); got != "udb-1" {
			t.Fatalf("unexpected db id: %s", got)
		}
		_, _ = w.Write([]byte(`{"Action":"CreateUDBUserResponse","RetCode":0}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	res, err := driver.CreateAccount(context.Background(), "udb-1")
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
		_ = r.ParseForm()
		if r.Form.Get("Action") != "DeleteUDBUser" {
			t.Fatalf("unexpected action: %s", r.Form.Get("Action"))
		}
		_, _ = w.Write([]byte(`{"Action":"DeleteUDBUserResponse","RetCode":0}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	res, err := driver.DeleteAccount(context.Background(), "udb-1")
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if res.Username != "ctkuser" {
		t.Errorf("unexpected username: %s", res.Username)
	}
}

func TestCreateAccountRejectsMissingConfig(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{})
	driver := newDriver("http://example.invalid")
	if _, err := driver.CreateAccount(context.Background(), "udb-1"); err == nil {
		t.Fatalf("expected error for missing rds-account-check")
	}
}

func TestDeleteAccountPropagatesError(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Action":"DeleteUDBUserResponse","RetCode":1011,"Message":"user not found"}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	if _, err := driver.DeleteAccount(context.Background(), "udb-1"); err == nil {
		t.Fatalf("expected error from DeleteAccount")
	} else if !strings.Contains(err.Error(), "user not found") {
		t.Errorf("expected error to mention user not found, got %v", err)
	}
}
