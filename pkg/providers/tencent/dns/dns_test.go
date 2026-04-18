package dns

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

func TestGetDomainsPaginatesAndMapsRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "DescribeDomainList":
			body := readBody(t, r)
			switch body {
			case `{"Limit":3000}`:
				_, _ = w.Write([]byte(`{"Response":{"DomainCountInfo":{"DomainTotal":3},"DomainList":[{"Name":"example.com","Status":"ENABLE","DNSStatus":""},{"Name":"skip.com","Status":"PAUSE","DNSStatus":""}]}}`))
			case `{"Offset":2,"Limit":3000}`:
				_, _ = w.Write([]byte(`{"Response":{"DomainCountInfo":{"DomainTotal":3},"DomainList":[{"Name":"example.org","Status":"ENABLE","DNSStatus":"DNSERROR"}]}}`))
			default:
				t.Fatalf("unexpected domain list body: %s", body)
			}
		case "DescribeRecordList":
			body := readBody(t, r)
			switch body {
			case `{"Domain":"example.com","Limit":3000,"ErrorOnEmpty":"no"}`:
				_, _ = w.Write([]byte(`{"Response":{"RecordCountInfo":{"TotalCount":2},"RecordList":[{"Name":"@","Type":"A","Value":"1.1.1.1","Status":"ENABLE"}]}}`))
			case `{"Domain":"example.com","Offset":1,"Limit":3000,"ErrorOnEmpty":"no"}`:
				_, _ = w.Write([]byte(`{"Response":{"RecordCountInfo":{"TotalCount":2},"RecordList":[{"Name":"www","Type":"CNAME","Value":"target.example.com","Status":"ENABLE"}]}}`))
			default:
				t.Fatalf("unexpected record list body: %s", body)
			}
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := Driver{
		Credential: auth.New("ak", "sk", ""),
		Region:     "all",
		clientOptions: []api.Option{
			api.WithBaseURL(server.URL),
			api.WithClock(func() time.Time { return time.Unix(1776458501, 0).UTC() }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		},
	}

	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains() error = %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("unexpected domain count: %d", len(domains))
	}
	if domains[0].DomainName != "example.com" {
		t.Fatalf("unexpected domain name: %+v", domains[0])
	}
	if len(domains[0].Records) != 2 {
		t.Fatalf("unexpected record count: %d", len(domains[0].Records))
	}
	if domains[0].Records[1].Value != "target.example.com" {
		t.Fatalf("unexpected second record: %+v", domains[0].Records[1])
	}
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	defer r.Body.Close()
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(buf)
}
