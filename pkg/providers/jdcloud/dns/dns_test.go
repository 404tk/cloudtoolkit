package dns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
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

func TestGetDomainsListsDomainsAndRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/domain") && !strings.Contains(r.URL.Path, "/domain/"):
			_, _ = w.Write([]byte(`{"requestId":"r1","result":{"dataList":[
  {"id":1001,"domainName":"ctk.example.com","resolvingStatus":"2","jcloudNs":true},
  {"id":1002,"domainName":"ctk-internal.example","resolvingStatus":"2","jcloudNs":true}
],"currentCount":2,"totalCount":2,"totalPage":1}}`))
		case strings.HasSuffix(r.URL.Path, "/1001/ResourceRecord"):
			_, _ = w.Write([]byte(`{"requestId":"r2","result":{"dataList":[
  {"id":11,"hostRecord":"@","type":"A","hostValue":"198.51.100.10","ttl":300,"resolvingStatus":"2"},
  {"id":12,"hostRecord":"www","type":"CNAME","hostValue":"ctk.example.com","ttl":60,"resolvingStatus":"2"},
  {"id":13,"hostRecord":"@","type":"MX","hostValue":"10 mx1","ttl":300,"resolvingStatus":"4"},
  {"id":14,"hostRecord":"@","type":"SOA","hostValue":"ns1.jdcloud.com hostmaster 1 7200 3600 604800 86400","ttl":3600,"resolvingStatus":"2"}
],"currentCount":4,"totalCount":4,"totalPage":1}}`))
		case strings.HasSuffix(r.URL.Path, "/1002/ResourceRecord"):
			_, _ = w.Write([]byte(`{"requestId":"r3","result":{"dataList":[
  {"id":21,"hostRecord":"db","type":"A","hostValue":"10.0.0.10","ttl":60,"resolvingStatus":"2"}
],"currentCount":1,"totalCount":1,"totalPage":1}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	if domains[0].DomainName != "ctk.example.com" {
		t.Errorf("unexpected first domain: %+v", domains[0])
	}
	// SOA filtered out: A + CNAME + MX remain.
	if len(domains[0].Records) != 3 {
		t.Fatalf("expected 3 records on first domain (SOA filtered), got %d", len(domains[0].Records))
	}
	want := []struct {
		rr, recordType, value, status string
	}{
		{"@", "A", "198.51.100.10", "Enable"},
		{"www", "CNAME", "ctk.example.com", "Enable"},
		{"@", "MX", "10 mx1", "Pause"},
	}
	for i, w := range want {
		got := domains[0].Records[i]
		if got.RR != w.rr || got.Type != w.recordType || got.Value != w.value || got.Status != w.status {
			t.Errorf("record[%d] = %+v, want %+v", i, got, w)
		}
	}
	if len(domains[1].Records) != 1 || domains[1].Records[0].Status != "Enable" {
		t.Errorf("unexpected second domain records: %+v", domains[1].Records)
	}
}

// TestGetDomainsWalksRecordPagination exercises the driver's per-zone RR
// pagination loop. The replay handler honours pageNumber/pageSize, so a
// fixture larger than one page must surface every record exactly once.
func TestGetDomainsWalksRecordPagination(t *testing.T) {
	totalRecords := 150
	requested := struct {
		sync.Mutex
		pages map[string]int
	}{pages: map[string]int{}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/domain") && !strings.Contains(r.URL.Path, "/domain/"):
			_, _ = w.Write([]byte(`{"requestId":"d1","result":{"dataList":[
  {"id":2001,"domainName":"big.example.com","resolvingStatus":"2","jcloudNs":true}
],"currentCount":1,"totalCount":1,"totalPage":1}}`))
		case strings.HasSuffix(r.URL.Path, "/2001/ResourceRecord"):
			page := r.URL.Query().Get("pageNumber")
			size := r.URL.Query().Get("pageSize")
			requested.Lock()
			requested.pages[page]++
			requested.Unlock()
			if size != "100" {
				t.Errorf("pageSize=100 expected, got %q", size)
			}
			records := pageRecords(page, totalRecords, 100)
			payload, _ := json.Marshal(map[string]any{
				"requestId": "d2",
				"result": map[string]any{
					"dataList":     records,
					"currentCount": len(records),
					"totalCount":   totalRecords,
					"totalPage":    2,
				},
			})
			_, _ = w.Write(payload)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains: %v", err)
	}
	if len(domains) != 1 || len(domains[0].Records) != totalRecords {
		t.Fatalf("expected %d records across 1 domain, got %d domains / %d records",
			totalRecords, len(domains), func() int {
				if len(domains) == 0 {
					return 0
				}
				return len(domains[0].Records)
			}())
	}
	// Both pages must have been visited exactly once.
	requested.Lock()
	defer requested.Unlock()
	if requested.pages["1"] != 1 || requested.pages["2"] != 1 {
		t.Errorf("pagination call counts: %+v (want page 1 = 1, page 2 = 1)", requested.pages)
	}
}

// pageRecords returns the JSONable RRInfo slice for a given page of a fixed
// total. RR types alternate A / AAAA so the type filter stays a no-op.
func pageRecords(page string, total, pageSize int) []map[string]any {
	if pageSize <= 0 {
		return nil
	}
	pageNumber := 1
	if page == "2" {
		pageNumber = 2
	}
	start := (pageNumber - 1) * pageSize
	if start >= total {
		return nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	out := make([]map[string]any, 0, end-start)
	for i := start; i < end; i++ {
		recordType := "A"
		if i%2 == 1 {
			recordType = "AAAA"
		}
		out = append(out, map[string]any{
			"id":              i + 1,
			"hostRecord":      "host-" + strconv.Itoa(i),
			"type":            recordType,
			"hostValue":       "203.0.113." + strconv.Itoa(i%254+1),
			"ttl":             60,
			"resolvingStatus": "2",
		})
	}
	return out
}

func TestGetDomainsPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"requestId":"r-err","error":{"code":"AccessDenied","message":"forbidden"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	_, err := driver.GetDomains(context.Background())
	if err == nil {
		t.Fatal("expected error when DescribeDomains fails")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("expected AccessDenied in err, got %v", err)
	}
}

func TestGetDomainsHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"requestId":"r2","result":{"dataList":[],"currentCount":0,"totalCount":0,"totalPage":0}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains: %v", err)
	}
	if len(domains) != 0 {
		t.Errorf("expected 0 domains, got %d", len(domains))
	}
}
