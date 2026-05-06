package insights

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
)

func newTestClient(server *httptest.Server) *azapi.Client {
	cred := auth.New("client", "secret", "tenant", "sub", "")
	ts := auth.NewTokenSource(cred, server.Client())
	auth.SetCachedToken(ts, auth.Token{AccessToken: "demo", ExpiresAt: time.Now().Add(time.Hour)})
	endpoints := cloud.For(cred.Cloud)
	return azapi.NewClient(ts, endpoints, azapi.WithBaseURL(server.URL), azapi.WithHTTPClient(server.Client()))
}

func TestDumpEventsParses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/Microsoft.Insights/eventtypes/management/values") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		filter := r.URL.Query().Get("$filter")
		if !strings.Contains(filter, "eventTimestamp ge") {
			t.Fatalf("expected eventTimestamp filter, got: %s", filter)
		}
		_, _ = w.Write([]byte(`{"value":[{"eventDataId":"e1","operationName":{"value":"Microsoft.Authorization/roleAssignments/write","localizedValue":"Create role assignment"},"eventTimestamp":"2026-04-22T09:11:00.0000000Z","caller":"admin@contoso.com","httpRequest":{"clientIpAddress":"1.1.1.1"},"resourceId":"/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Authorization/roleAssignments/r1","status":{"value":"Succeeded","localizedValue":"Succeeded"}},{"eventDataId":"e2","operationName":{"value":"Microsoft.Storage/storageAccounts/blobServices/containers/write","localizedValue":"Update blob container"},"eventTimestamp":"2026-04-22T09:14:00Z","caller":"app","status":{"value":"Failed"}}]}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server), SubscriptionIDs: []string{"sub"}}
	events, err := driver.DumpEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Id != "e1" || events[0].Name != "Create role assignment" {
		t.Errorf("unexpected first event: %+v", events[0])
	}
	if events[1].Status != "Failed" {
		t.Errorf("unexpected second status: %+v", events[1])
	}
}

func TestDumpEventsParsesTimeWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter := r.URL.Query().Get("$filter")
		if !strings.Contains(filter, "2023-11-14T22:13:20Z") {
			t.Fatalf("expected start ISO in filter, got: %s", filter)
		}
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server), SubscriptionIDs: []string{"sub"}}
	if _, err := driver.DumpEvents(context.Background(), "1700000000:1700003600"); err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
}

func TestDumpEventsRejectsNoSubscription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call API without subscription")
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server), SubscriptionIDs: nil}
	if _, err := driver.DumpEvents(context.Background(), ""); err == nil {
		t.Fatalf("expected error for missing subscription")
	}
}

func TestHandleEventsReturnsUnsupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server), SubscriptionIDs: []string{"sub"}}
	if _, err := driver.HandleEvents(context.Background(), ""); err == nil {
		t.Fatalf("expected error from HandleEvents")
	}
}
