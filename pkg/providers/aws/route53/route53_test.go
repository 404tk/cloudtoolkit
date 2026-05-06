package route53

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func newRoute53TestDriver(baseURL string) *Driver {
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
	}
}

const sampleListHostedZones = `<?xml version="1.0"?>
<ListHostedZonesResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <HostedZones>
    <HostedZone>
      <Id>/hostedzone/Z2EXAMPLEPUBLIC</Id>
      <Name>example.com.</Name>
      <CallerReference>ctk-test-public</CallerReference>
      <Config>
        <Comment>example public zone</Comment>
        <PrivateZone>false</PrivateZone>
      </Config>
      <ResourceRecordSetCount>4</ResourceRecordSetCount>
    </HostedZone>
    <HostedZone>
      <Id>/hostedzone/Z2EXAMPLEPRIVATE</Id>
      <Name>internal.example.</Name>
      <CallerReference>ctk-test-private</CallerReference>
      <Config>
        <Comment>example private zone</Comment>
        <PrivateZone>true</PrivateZone>
      </Config>
      <ResourceRecordSetCount>2</ResourceRecordSetCount>
    </HostedZone>
  </HostedZones>
  <IsTruncated>false</IsTruncated>
  <MaxItems>100</MaxItems>
</ListHostedZonesResponse>`

const sampleRRSetsPublic = `<?xml version="1.0"?>
<ListResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ResourceRecordSets>
    <ResourceRecordSet>
      <Name>example.com.</Name>
      <Type>A</Type>
      <TTL>300</TTL>
      <ResourceRecords>
        <ResourceRecord><Value>198.51.100.10</Value></ResourceRecord>
        <ResourceRecord><Value>198.51.100.11</Value></ResourceRecord>
      </ResourceRecords>
    </ResourceRecordSet>
    <ResourceRecordSet>
      <Name>www.example.com.</Name>
      <Type>CNAME</Type>
      <TTL>60</TTL>
      <ResourceRecords>
        <ResourceRecord><Value>example.com.</Value></ResourceRecord>
      </ResourceRecords>
    </ResourceRecordSet>
    <ResourceRecordSet>
      <Name>example.com.</Name>
      <Type>NS</Type>
      <TTL>172800</TTL>
      <ResourceRecords>
        <ResourceRecord><Value>ns-1.example.com.</Value></ResourceRecord>
      </ResourceRecords>
    </ResourceRecordSet>
    <ResourceRecordSet>
      <Name>cdn.example.com.</Name>
      <Type>A</Type>
      <AliasTarget>
        <HostedZoneId>Z2FDTNDATAQYW2</HostedZoneId>
        <DNSName>d111111abcdef8.cloudfront.net.</DNSName>
        <EvaluateTargetHealth>false</EvaluateTargetHealth>
      </AliasTarget>
    </ResourceRecordSet>
  </ResourceRecordSets>
  <IsTruncated>false</IsTruncated>
  <MaxItems>100</MaxItems>
</ListResourceRecordSetsResponse>`

const sampleRRSetsPrivate = `<?xml version="1.0"?>
<ListResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ResourceRecordSets>
    <ResourceRecordSet>
      <Name>db.internal.example.</Name>
      <Type>A</Type>
      <TTL>60</TTL>
      <ResourceRecords>
        <ResourceRecord><Value>10.0.0.10</Value></ResourceRecord>
      </ResourceRecords>
    </ResourceRecordSet>
  </ResourceRecordSets>
  <IsTruncated>false</IsTruncated>
  <MaxItems>100</MaxItems>
</ListResourceRecordSetsResponse>`

// route53TestServer routes the small set of paths the driver hits.
func route53TestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/2013-04-01/hostedzone":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(sampleListHostedZones))
		case strings.HasPrefix(path, "/2013-04-01/hostedzone/Z2EXAMPLEPUBLIC/rrset"):
			_, _ = w.Write([]byte(sampleRRSetsPublic))
		case strings.HasPrefix(path, "/2013-04-01/hostedzone/Z2EXAMPLEPRIVATE/rrset"):
			_, _ = w.Write([]byte(sampleRRSetsPrivate))
		default:
			http.Error(w, "unhandled path: "+path, http.StatusNotFound)
		}
	}))
}

func TestGetDomainsListsZonesAndRecords(t *testing.T) {
	server := route53TestServer(t)
	defer server.Close()

	driver := newRoute53TestDriver(server.URL)
	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}

	pub := domains[0]
	if pub.DomainName != "example.com" {
		t.Errorf("unexpected public domain name: %q", pub.DomainName)
	}
	// 2 A values + 1 CNAME + 1 alias A = 4 surfaced records; NS is filtered.
	if len(pub.Records) != 4 {
		t.Fatalf("expected 4 records on public zone, got %d (%v)", len(pub.Records), pub.Records)
	}
	var sawAlias, sawA1, sawCNAME bool
	for _, rec := range pub.Records {
		switch {
		case rec.Type == "A" && rec.Value == "198.51.100.10":
			sawA1 = true
		case rec.Type == "CNAME" && rec.Value == "example.com.":
			sawCNAME = true
		case rec.Type == "A" && strings.HasPrefix(rec.Value, "ALIAS "):
			sawAlias = true
		}
	}
	if !sawA1 || !sawCNAME || !sawAlias {
		t.Errorf("missing expected record kinds: A=%v CNAME=%v ALIAS=%v in %+v", sawA1, sawCNAME, sawAlias, pub.Records)
	}

	priv := domains[1]
	if priv.DomainName != "internal.example" {
		t.Errorf("unexpected private domain name: %q", priv.DomainName)
	}
	if len(priv.Records) != 1 || priv.Records[0].Value != "10.0.0.10" {
		t.Errorf("unexpected private records: %+v", priv.Records)
	}
}

func TestGetDomainsContinuesPastPerZoneError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		path := r.URL.Path
		switch {
		case path == "/2013-04-01/hostedzone":
			_, _ = w.Write([]byte(sampleListHostedZones))
		case strings.HasPrefix(path, "/2013-04-01/hostedzone/Z2EXAMPLEPUBLIC/rrset"):
			http.Error(w, `<ErrorResponse><Error><Code>AccessDenied</Code><Message>denied</Message></Error></ErrorResponse>`, http.StatusForbidden)
		case strings.HasPrefix(path, "/2013-04-01/hostedzone/Z2EXAMPLEPRIVATE/rrset"):
			_, _ = w.Write([]byte(sampleRRSetsPrivate))
		default:
			http.Error(w, "unhandled path: "+path, http.StatusNotFound)
		}
	}))
	defer server.Close()

	driver := newRoute53TestDriver(server.URL)
	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains returned fatal error %v; wanted partial recovery", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains even after one rrset failure, got %d", len(domains))
	}
	if len(domains[0].Records) != 0 {
		t.Errorf("expected denied zone to have 0 records, got %d", len(domains[0].Records))
	}
	if len(domains[1].Records) != 1 {
		t.Errorf("expected private zone records preserved, got %d", len(domains[1].Records))
	}
}

func TestGetDomainsRejectsHostedZoneFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `<ErrorResponse><Error><Code>InvalidClientTokenId</Code><Message>bad creds</Message></Error></ErrorResponse>`, http.StatusForbidden)
	}))
	defer server.Close()

	driver := newRoute53TestDriver(server.URL)
	_, err := driver.GetDomains(context.Background())
	if err == nil {
		t.Fatalf("expected error when ListHostedZones fails")
	}
	if !strings.Contains(err.Error(), "InvalidClientTokenId") {
		t.Errorf("error should propagate InvalidClientTokenId, got: %v", err)
	}
}
