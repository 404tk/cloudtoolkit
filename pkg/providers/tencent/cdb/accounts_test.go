package cdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
)

func TestCreateAccountSendsExpectedPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Get("X-TC-Action")
		body := readBody(t, r)
		if !strings.Contains(body, `"InstanceId":"cdb-1"`) || !strings.Contains(body, `"User":"ctkuser"`) {
			t.Fatalf("unexpected body: %s", body)
		}
		_, _ = w.Write([]byte(`{"Response":{"AsyncRequestId":"async-1","RequestId":"r1"}}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "ap-guangzhou")
	res, err := driver.CreateAccount(context.Background(), "cdb-1")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if captured != "CreateAccounts" {
		t.Errorf("unexpected action: %s", captured)
	}
	if res.Username != "ctkuser" || res.Password != "Ctk!Pwd2026" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestCreateAccountRejectsMissingConfig(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{})
	driver := newTestDriver("http://example.invalid", "ap-guangzhou")
	if _, err := driver.CreateAccount(context.Background(), "cdb-1"); err == nil {
		t.Fatalf("expected error for missing rds-account-check config")
	}
}

func TestDeleteAccountSendsExpectedPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-TC-Action") != "DeleteAccounts" {
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
		body := readBody(t, r)
		if !strings.Contains(body, `"User":"ctkuser"`) {
			t.Fatalf("unexpected body: %s", body)
		}
		_, _ = w.Write([]byte(`{"Response":{"AsyncRequestId":"async-2","RequestId":"r2"}}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "ap-guangzhou")
	res, err := driver.DeleteAccount(context.Background(), "cdb-1")
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if res.Username != "ctkuser" {
		t.Errorf("unexpected username: %s", res.Username)
	}
}

func TestDeleteAccountPropagatesAPIError(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"Response":{"Error":{"Code":"ResourceNotFound.Account","Message":"account not found"},"RequestId":"r1"}}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "ap-guangzhou")
	if _, err := driver.DeleteAccount(context.Background(), "cdb-1"); err == nil {
		t.Fatalf("expected error from DeleteAccount")
	}
}
