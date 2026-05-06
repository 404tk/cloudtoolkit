package rds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
)

func newDriver(baseURL string) *Driver {
	return &Driver{
		Client: api.NewClient(
			auth.New("AKID", "SECRET", ""),
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		),
		Region:        "us-east-1",
		DefaultRegion: "us-east-1",
	}
}

func parseForm(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatalf("ParseForm: %v", err)
	}
	return r.PostForm
}

func TestCreateAccountSendsRotationPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := parseForm(t, r)
		if got := values.Get("Action"); got != "ModifyDBInstance" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := values.Get("DBInstanceIdentifier"); got != "rds-1" {
			t.Fatalf("unexpected instance: %s", got)
		}
		if got := values.Get("MasterUserPassword"); got != "Ctk!Pwd2026" {
			t.Fatalf("unexpected password: %s", got)
		}
		_, _ = w.Write([]byte(`<ModifyDBInstanceResponse><ModifyDBInstanceResult><DBInstance><DBInstanceIdentifier>rds-1</DBInstanceIdentifier><DBInstanceStatus>modifying</DBInstanceStatus><MasterUsername>admin</MasterUsername></DBInstance></ModifyDBInstanceResult><ResponseMetadata><RequestId>r1</RequestId></ResponseMetadata></ModifyDBInstanceResponse>`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	res, err := driver.CreateAccount(context.Background(), "rds-1")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if res.Username != "admin" || res.Password != "Ctk!Pwd2026" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestDeleteAccountSendsRandomPassword(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = parseForm(t, r).Get("MasterUserPassword")
		_, _ = w.Write([]byte(`<ModifyDBInstanceResponse><ModifyDBInstanceResult><DBInstance><DBInstanceIdentifier>rds-1</DBInstanceIdentifier><DBInstanceStatus>modifying</DBInstanceStatus><MasterUsername>admin</MasterUsername></DBInstance></ModifyDBInstanceResult><ResponseMetadata><RequestId>r1</RequestId></ResponseMetadata></ModifyDBInstanceResponse>`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	res, err := driver.DeleteAccount(context.Background(), "rds-1")
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if res.Username != "admin" {
		t.Errorf("unexpected username: %s", res.Username)
	}
	if captured == "" || captured == "Ctk!Pwd2026" {
		t.Errorf("expected random password (not config password), got %q", captured)
	}
}

func TestCreateAccountRejectsMissingConfig(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{})
	driver := newDriver("http://example.invalid")
	if _, err := driver.CreateAccount(context.Background(), "rds-1"); err == nil {
		t.Fatalf("expected error for missing rds-account-check")
	}
}

func TestDeleteAccountPropagatesAPIError(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<ErrorResponse><Error><Type>Sender</Type><Code>DBInstanceNotFound</Code><Message>not found</Message></Error><RequestId>r1</RequestId></ErrorResponse>`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	if _, err := driver.DeleteAccount(context.Background(), "rds-1"); err == nil {
		t.Fatalf("expected error from DeleteAccount")
	} else if !strings.Contains(err.Error(), "DBInstanceNotFound") {
		t.Errorf("expected DBInstanceNotFound, got %v", err)
	}
}
