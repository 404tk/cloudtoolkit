package rds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
)

func newTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func TestCreateAccountSendsExpectedPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("Action") != "CreateDBAccount" {
			t.Fatalf("unexpected action: %s", query.Get("Action"))
		}
		body := readBody(t, r)
		if !strings.Contains(body, `"InstanceId":"mysql-demo"`) || !strings.Contains(body, `"AccountName":"ctkuser"`) {
			t.Fatalf("unexpected body: %s", body)
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	res, err := driver.CreateAccount(context.Background(), "mysql-demo")
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
		if r.URL.Query().Get("Action") != "DeleteDBAccount" {
			t.Fatalf("unexpected action: %s", r.URL.Query().Get("Action"))
		}
		body := readBody(t, r)
		if !strings.Contains(body, `"AccountName":"ctkuser"`) {
			t.Fatalf("unexpected body: %s", body)
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	res, err := driver.DeleteAccount(context.Background(), "mysql-demo")
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if res.Username != "ctkuser" {
		t.Errorf("unexpected username: %s", res.Username)
	}
}

func TestServiceForInstance(t *testing.T) {
	cases := map[string]string{
		"mysql-x":   api.ServiceRDSMySQL,
		"postgres-y": api.ServiceRDSPostgreSQL,
		"pg-z":      api.ServiceRDSPostgreSQL,
		"mssql-q":   api.ServiceRDSMSSQL,
		"unknown":   api.ServiceRDSMySQL,
	}
	for in, want := range cases {
		if got := serviceForInstance(in); got != want {
			t.Errorf("serviceForInstance(%q)=%s, want %s", in, got, want)
		}
	}
}

func TestCreateAccountRejectsMissingConfig(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{})
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-beijing"}
	if _, err := driver.CreateAccount(context.Background(), "mysql-demo"); err == nil {
		t.Fatalf("expected error for missing rds-account-check")
	}
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	defer r.Body.Close()
	buf := make([]byte, 0, 256)
	tmp := make([]byte, 256)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return string(buf)
}
