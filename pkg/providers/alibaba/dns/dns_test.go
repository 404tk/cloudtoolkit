package dns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

func TestGetDomainsWithPagination(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("Action")
		switch action {
		case "DescribeDomains":
			switch r.URL.Query().Get("PageNumber") {
			case "1":
				_, _ = w.Write([]byte(`{"TotalCount":2,"PageSize":1,"PageNumber":1,"Domains":{"Domain":[{"DomainName":"example.com"}]}}`))
			case "2":
				_, _ = w.Write([]byte(`{"TotalCount":2,"PageSize":1,"PageNumber":2,"Domains":{"Domain":[{"DomainName":"example.org"}]}}`))
			default:
				t.Fatalf("unexpected domain page: %s", r.URL.Query().Get("PageNumber"))
			}
		case "DescribeDomainRecords":
			switch r.URL.Query().Get("DomainName") {
			case "example.com":
				_, _ = w.Write([]byte(`{"TotalCount":1,"PageSize":1,"PageNumber":1,"DomainRecords":{"Record":[{"RR":"@","Type":"A","Value":"1.1.1.1","Status":"ENABLE"}]}}`))
			case "example.org":
				_, _ = w.Write([]byte(`{"TotalCount":1,"PageSize":1,"PageNumber":1,"DomainRecords":{"Record":[{"RR":"www","Type":"CNAME","Value":"target.example.com","Status":"ENABLE"}]}}`))
			default:
				t.Fatalf("unexpected domain name: %s", r.URL.Query().Get("DomainName"))
			}
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := Driver{
		Cred:   aliauth.New("ak", "sk", ""),
		Region: "all",
		clientOptions: []api.Option{
			api.WithBaseURL(server.URL),
			api.WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
			api.WithNonce(func() string { return "nonce" }),
		},
	}

	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("get domains: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("unexpected domain count: %d", len(domains))
	}
	if domains[0].DomainName != "example.com" || len(domains[0].Records) != 1 {
		t.Fatalf("unexpected first domain: %+v", domains[0])
	}
	if domains[1].DomainName != "example.org" || domains[1].Records[0].Value != "target.example.com" {
		t.Fatalf("unexpected second domain: %+v", domains[1])
	}
}
