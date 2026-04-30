package storage

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
)

func TestDriverGetStoragesFollowsPagination(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/subscriptions/sub-1/providers/Microsoft.Storage/storageAccounts":
			_, _ = w.Write([]byte(`{"value":[{"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1","name":"acct-1","location":"eastasia"}]}`))
		case "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1/blobServices":
			_, _ = w.Write([]byte(`{"value":[{"name":"default"}]}`))
		case "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1/blobServices/default/containers":
			if r.URL.Query().Get("page") == "2" {
				_, _ = w.Write([]byte(`{"value":[{"name":"container-2"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"value":[{"name":"container-1"}],"nextLink":"` + server.URL + `/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1/blobServices/default/containers?page=2"}`))
		default:
			t.Fatalf("unexpected path: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server, []string{"sub-1"})
	got, err := driver.GetStorages(context.Background())
	if err != nil {
		t.Fatalf("GetStorages failed: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("unexpected storage count: %d", len(got))
	}
	if got[0].BucketName != "default(Blob Service)" || got[1].BucketName != "container-1(Blob Container)" || got[2].BucketName != "container-2(Blob Container)" {
		t.Fatalf("unexpected storages: %+v", got)
	}
}

func TestListBlobContainersIncludesPublicAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/subscriptions/sub-1/providers/Microsoft.Storage/storageAccounts":
			_, _ = w.Write([]byte(`{"value":[{"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1","name":"acct-1","location":"eastasia"}]}`))
		case "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1/blobServices/default/containers":
			_, _ = w.Write([]byte(`{"value":[{"name":"public","properties":{"publicAccess":"Blob"}},{"name":"private","properties":{"publicAccess":"None"}},{"name":"unset"}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server, []string{"sub-1"})
	got, err := driver.ListBlobContainers(context.Background())
	if err != nil {
		t.Fatalf("ListBlobContainers: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 containers, got %d", len(got))
	}
	wantLevels := map[string]string{"public": "Blob", "private": "None", "unset": "None"}
	for _, c := range got {
		if wantLevels[c.Name] != c.PublicAccess {
			t.Errorf("container %s: got level %q want %q", c.Name, c.PublicAccess, wantLevels[c.Name])
		}
	}
}

func TestSetContainerACLPATCHesPublicAccess(t *testing.T) {
	var lastBody []byte
	var lastMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1/blobServices/default/containers/audit":
			lastMethod = r.Method
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			lastBody = body
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"audit","properties":{"publicAccess":"Blob"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server, []string{"sub-1"})
	if err := driver.SetContainerACL(context.Background(), "sub-1", "rg-1", "acct-1", "audit", "Blob"); err != nil {
		t.Fatalf("SetContainerACL: %v", err)
	}
	if lastMethod != http.MethodPatch {
		t.Errorf("expected PATCH; got %s", lastMethod)
	}
	var got azapi.BlobContainerPatchRequest
	if err := json.Unmarshal(lastBody, &got); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if got.Properties.PublicAccess != "Blob" {
		t.Errorf("expected publicAccess=Blob; got %q", got.Properties.PublicAccess)
	}
}

func TestSetContainerACLRejectsUnknownLevel(t *testing.T) {
	driver := &Driver{}
	if err := driver.SetContainerACL(context.Background(), "s", "g", "a", "c", "Open"); err == nil {
		t.Fatal("expected error on bogus level")
	}
}

func TestNormalizePublicAccess(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: "None"},
		{in: "none", want: "None"},
		{in: "None", want: "None"},
		{in: "blob", want: "Blob"},
		{in: "Blob", want: "Blob"},
		{in: "container", want: "Container"},
		{in: "Open", wantErr: true},
	}
	for _, tc := range cases {
		got, err := normalizePublicAccess(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("normalizePublicAccess(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizePublicAccess(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("normalizePublicAccess(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

// --- helpers ---

func newTestDriver(t *testing.T, server *httptest.Server, subscriptions []string) *Driver {
	t.Helper()
	httpClient := server.Client()
	httpClient.Transport = tokenRewriteTransport{base: httpClient.Transport, target: mustParseURL(t, server.URL)}
	ts := auth.NewTokenSource(auth.New("client", "secret", "tenant", "", auth.CloudPublic), httpClient)
	client := azapi.NewClient(ts, cloud.For(auth.CloudPublic), azapi.WithHTTPClient(httpClient), azapi.WithBaseURL(server.URL))
	return &Driver{Client: client, SubscriptionIDs: subscriptions}
}

type tokenRewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (rt tokenRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "login.microsoftonline.com" {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = rt.target.Scheme
		clone.URL.Host = rt.target.Host
		clone.Host = rt.target.Host
		return rt.base.RoundTrip(clone)
	}
	return rt.base.RoundTrip(req)
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return u
}
