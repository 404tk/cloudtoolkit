package actiontrail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
)

func newTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		api.WithNonceFunc(func() string { return "ebf8b26d-c3be-402f-9f10-f8b6573fd823" }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func TestDumpEventsParses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/regions/cn-north-1/events" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			RegionID         string `json:"regionId"`
			PageNumber       int    `json:"pageNumber"`
			PageSize         int    `json:"pageSize"`
			LookupAttributes string `json:"lookupAttributes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.RegionID != "cn-north-1" || body.PageNumber != 1 || body.PageSize != 20 {
			t.Fatalf("unexpected request body: %+v", body)
		}
		var lookup map[string]string
		if err := json.Unmarshal([]byte(body.LookupAttributes), &lookup); err != nil {
			t.Fatalf("decode lookupAttributes: %v", err)
		}
		if lookup["accessKeyId"] != "AKID" {
			t.Fatalf("unexpected lookupAttributes: %+v", lookup)
		}
		_, _ = w.Write([]byte(`{"requestId":"r1","result":{"pageSize":20,"pageNumber":1,"totalNumber":2,"events":[{"eventId":"e1","eventName":"CreateSubUser","eventTime":1700000000,"ip":"203.0.113.62","accessKeyId":"AKID","resources":[{"resourceName":"subUser/audit"}]},{"eventId":"e2","eventName":"DeleteAccessKey","eventTime":1700000300000,"errorCode":"AccessDenied"}]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1", AccessKey: "AKID"}
	events, err := driver.DumpEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Id != "e1" || events[0].Name != "CreateSubUser" {
		t.Errorf("unexpected first event: %+v", events[0])
	}
	if events[0].SourceIp != "203.0.113.62" || events[0].Affected != "subUser/audit" {
		t.Errorf("unexpected first event mapping: %+v", events[0])
	}
	if events[0].Time != "2023-11-14T22:13:20Z" || events[1].Time != "2023-11-14T22:18:20Z" {
		t.Errorf("unexpected event time mapping: %+v", events)
	}
	if events[1].Status != "Failed" {
		t.Errorf("unexpected second status: %+v", events[1])
	}
}

func TestDumpEventsParsesTimeWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			StartTime int64 `json:"startTime"`
			EndTime   int64 `json:"endTime"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.StartTime != 1700000000 || body.EndTime != 1700003600 {
			t.Fatalf("unexpected window: %d/%d", body.StartTime, body.EndTime)
		}
		_, _ = w.Write([]byte(`{"requestId":"r1","result":{"pageSize":20,"pageNumber":1,"totalNumber":0,"events":[]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1", AccessKey: "AKID"}
	if _, err := driver.DumpEvents(context.Background(), "1700000000:1700003600"); err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
}

func TestDumpEventsLimitsDefaultResultCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseEvents := make([]map[string]any, 25)
		for i := range responseEvents {
			responseEvents[i] = map[string]any{
				"eventId":   fmt.Sprintf("e%02d", i),
				"eventName": "DemoEvent",
				"eventTime": 1700000000 + i,
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"requestId": "r1",
			"result": map[string]any{
				"pageSize":    20,
				"pageNumber":  1,
				"totalNumber": 25,
				"events":      responseEvents,
			},
		})
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1", AccessKey: "AKID"}
	events, err := driver.DumpEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if len(events) != 20 {
		t.Fatalf("expected default limit of 20 events, got %d", len(events))
	}
}

func TestDumpEventsRejectsMalformedWindow(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-north-1"}
	if _, err := driver.DumpEvents(context.Background(), "garbage"); err == nil {
		t.Fatalf("expected error for malformed window")
	}
}

func TestDumpEventsPropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"requestId":"r1","error":{"code":403,"message":"audit not enabled"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	_, err := driver.DumpEvents(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for 403")
	}
	if !strings.Contains(err.Error(), "audit not enabled") {
		t.Errorf("expected error to contain message, got %v", err)
	}
}

func TestHandleEventsReturnsUnsupported(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-north-1"}
	if _, err := driver.HandleEvents(context.Background(), ""); err == nil {
		t.Fatalf("expected error from HandleEvents")
	}
}
