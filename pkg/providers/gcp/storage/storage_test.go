package storage

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func TestGetBucketsPaginatesAndMapsStorage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeToken(w)
		case "/storage/v1/b":
			if got := r.URL.Query().Get("project"); got != "proj-1" {
				t.Fatalf("unexpected project: %s", got)
			}
			if got := r.URL.Query().Get("maxResults"); got != "200" {
				t.Fatalf("unexpected maxResults: %s", got)
			}
			switch r.URL.Query().Get("pageToken") {
			case "":
				_, _ = w.Write([]byte(`{"items":[{"name":"bucket-one","location":"US"}],"nextPageToken":"page-2"}`))
			case "page-2":
				_, _ = w.Write([]byte(`{"items":[{"name":"bucket-two","location":"EU"}]}`))
			default:
				t.Fatalf("unexpected pageToken: %s", r.URL.Query().Get("pageToken"))
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Projects: []string{"proj-1"}, Client: newStorageClient(t, server)}
	got, err := driver.GetBuckets(context.Background())
	if err != nil {
		t.Fatalf("GetBuckets: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(got))
	}
	if got[0].BucketName != "bucket-one" || got[0].Region != "US" {
		t.Fatalf("unexpected first bucket: %+v", got[0])
	}
	if got[1].BucketName != "bucket-two" || got[1].Region != "EU" {
		t.Fatalf("unexpected second bucket: %+v", got[1])
	}
}

func TestListAndTotalObjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeToken(w)
		case "/storage/v1/b/bucket-one/o":
			if got := r.URL.Query().Get("maxResults"); got != "200" {
				t.Fatalf("unexpected maxResults: %s", got)
			}
			switch r.URL.Query().Get("pageToken") {
			case "":
				_, _ = w.Write([]byte(`{"items":[{"bucket":"bucket-one","name":"a.txt","size":"7","updated":"2026-05-01T01:00:00Z","storageClass":"STANDARD"},{"bucket":"bucket-one","name":"b.txt","size":"11","updated":"2026-05-01T02:00:00Z","storageClass":"NEARLINE"}],"nextPageToken":"page-2"}`))
			case "page-2":
				_, _ = w.Write([]byte(`{"items":[{"bucket":"bucket-one","name":"c.txt","size":"13","updated":"2026-05-01T03:00:00Z","storageClass":"STANDARD"}]}`))
			default:
				t.Fatalf("unexpected pageToken: %s", r.URL.Query().Get("pageToken"))
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Projects: []string{"proj-1"}, Client: newStorageClient(t, server)}
	listed, err := driver.ListObjects(context.Background(), map[string]string{"bucket-one": ""})
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	if len(listed) != 1 || listed[0].ObjectCount != 3 {
		t.Fatalf("unexpected list result: %+v", listed)
	}
	if listed[0].Objects[0].Key != "a.txt" || listed[0].Objects[0].Size != 7 {
		t.Fatalf("unexpected first object: %+v", listed[0].Objects[0])
	}
}

func TestBucketACLAuditExposeAndUnexpose(t *testing.T) {
	public := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeToken(w)
		case "/storage/v1/b/bucket-one/iam":
			switch r.Method {
			case http.MethodGet:
				writePolicy(w, public)
			case http.MethodPut:
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				var policy api.GCSPolicy
				if err := json.Unmarshal(body, &policy); err != nil {
					t.Fatalf("unmarshal policy: %v", err)
				}
				public = policyHasMember(policy, allUsersMember)
				_, _ = w.Write([]byte(`{}`))
			default:
				t.Fatalf("unexpected method: %s", r.Method)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Projects: []string{"proj-1"}, Client: newStorageClient(t, server)}
	entries, err := driver.AuditBucketACL(context.Background(), "bucket-one")
	if err != nil {
		t.Fatalf("AuditBucketACL: %v", err)
	}
	if len(entries) != 1 || entries[0].Level != "Private" {
		t.Fatalf("expected private bucket, got %+v", entries)
	}
	level, err := driver.ExposeBucket(context.Background(), "bucket-one", "")
	if err != nil {
		t.Fatalf("ExposeBucket: %v", err)
	}
	if level != "Public" || !public {
		t.Fatalf("expected public state, level=%s public=%v", level, public)
	}
	entries, err = driver.AuditBucketACL(context.Background(), "bucket-one")
	if err != nil {
		t.Fatalf("AuditBucketACL after expose: %v", err)
	}
	if len(entries) != 1 || entries[0].Level != "Public" {
		t.Fatalf("expected public bucket, got %+v", entries)
	}
	if err := driver.UnexposeBucket(context.Background(), "bucket-one"); err != nil {
		t.Fatalf("UnexposeBucket: %v", err)
	}
	if public {
		t.Fatalf("expected allUsers to be removed")
	}
}

func newStorageClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	httpClient := server.Client()
	transport, err := testutil.RewriteHostsTransport(httpClient.Transport, server.URL, "storage.googleapis.com")
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

func writeToken(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
}

func writePolicy(w http.ResponseWriter, public bool) {
	if public {
		_, _ = w.Write([]byte(`{"version":3,"etag":"etag-1","bindings":[{"role":"roles/storage.objectViewer","members":["allUsers"]}]}`))
		return
	}
	_, _ = w.Write([]byte(`{"version":3,"etag":"etag-1","bindings":[{"role":"roles/storage.legacyBucketReader","members":["projectViewer:proj-1"]}]}`))
}

func policyHasMember(policy api.GCSPolicy, member string) bool {
	for _, binding := range policy.Bindings {
		for _, current := range binding.Members {
			if strings.EqualFold(current, member) {
				return true
			}
		}
	}
	return false
}
