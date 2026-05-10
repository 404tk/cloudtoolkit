package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
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

func TestDumpEventsParsesTimeWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			StartTime  int64 `json:"StartTime"`
			EndTime    int64 `json:"EndTime"`
			MaxResults int32 `json:"MaxResults"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.StartTime != 1700000000 {
			t.Fatalf("expected start unix 1700000000, got %d", body.StartTime)
		}
		if body.EndTime != 1700003600 {
			t.Fatalf("expected end unix 1700003600, got %d", body.EndTime)
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"},"Result":{"Trails":[],"NextToken":""}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	if _, err := driver.DumpEvents(context.Background(), "1700000000:1700003600"); err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
}

func TestDumpEventsRejectsMalformedWindow(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-beijing"}
	if _, err := driver.DumpEvents(context.Background(), "not-a-window"); err == nil {
		t.Fatalf("expected error for malformed window")
	}
}

func TestDumpEventsPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1","Error":{"Code":"Forbidden","Message":"audit not enabled"}}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	_, err := driver.DumpEvents(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for 403")
	}
	if !strings.Contains(err.Error(), "Forbidden") {
		t.Errorf("expected Forbidden in error, got %v", err)
	}
}

func TestHandleEventsReturnsUnsupported(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-beijing"}
	if _, err := driver.HandleEvents(context.Background(), ""); err == nil {
		t.Fatalf("expected error from HandleEvents")
	}
}
