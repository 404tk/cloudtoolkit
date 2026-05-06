package logging

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func newLoggingClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	httpClient := server.Client()
	transport, err := testutil.RewriteHostsTransport(httpClient.Transport, server.URL, "logging.googleapis.com")
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

func TestDumpEventsParses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/v2/entries:list") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body api.ListLogEntriesRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.Contains(body.Filter, "cloudaudit.googleapis.com") {
			t.Fatalf("expected cloudaudit filter, got: %s", body.Filter)
		}
		_, _ = w.Write([]byte(`{"entries":[{"insertId":"i1","logName":"projects/p/logs/cloudaudit.googleapis.com%2Factivity","timestamp":"2026-04-22T09:11:00Z","protoPayload":{"@type":"type.googleapis.com/google.cloud.audit.AuditLog","serviceName":"iam.googleapis.com","methodName":"google.iam.admin.v1.CreateServiceAccount","resourceName":"projects/p/serviceAccounts/x","authenticationInfo":{"principalEmail":"app@p.iam"},"requestMetadata":{"callerIp":"1.1.1.1"},"status":{"code":0}}},{"insertId":"i2","timestamp":"2026-04-22T09:14:00Z","protoPayload":{"methodName":"storage.buckets.update","status":{"code":7,"message":"PERMISSION_DENIED"}}}]}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newLoggingClient(t, server), Projects: []string{"proj-1"}}
	events, err := driver.DumpEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Id != "i1" || events[0].Name == "" {
		t.Errorf("unexpected first event: %+v", events[0])
	}
	if !strings.HasPrefix(events[1].Status, "Failed") {
		t.Errorf("expected Failed status for code=7, got %s", events[1].Status)
	}
}

func TestDumpEventsRejectsNoProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("API should not be called")
	}))
	defer server.Close()

	driver := &Driver{Client: newLoggingClient(t, server), Projects: nil}
	if _, err := driver.DumpEvents(context.Background(), ""); err == nil {
		t.Fatalf("expected error for missing project")
	}
}

func TestDumpEventsRejectsMalformedWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("API should not be called")
	}))
	defer server.Close()

	driver := &Driver{Client: newLoggingClient(t, server), Projects: []string{"proj-1"}}
	if _, err := driver.DumpEvents(context.Background(), "garbage"); err == nil {
		t.Fatalf("expected error for malformed window")
	}
}

func TestHandleEventsReturnsUnsupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	driver := &Driver{Client: newLoggingClient(t, server), Projects: []string{"proj-1"}}
	if _, err := driver.HandleEvents(context.Background(), ""); err == nil {
		t.Fatalf("expected error from HandleEvents")
	}
}
