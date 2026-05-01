package ufile

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ucloudapi "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

func newTestFileClient(t *testing.T, baseURL string) *FileClient {
	t.Helper()
	return NewFileClient(
		auth.New("ucloudpubkey-test", "ucloudprivkey-test", ""),
		WithFileHTTPClient(&http.Client{}),
		WithFileEndpointFormat(strings.TrimRight(baseURL, "/")),
		WithFileRetryPolicy(ucloudapi.RetryPolicy{MaxAttempts: 1}),
		WithFileClock(func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) }),
	)
}

func TestPrefixFileListSendsSignedRequest(t *testing.T) {
	var sawAuth string
	var sawDate string
	var sawQuery string
	var sawHost string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		sawDate = r.Header.Get("Date")
		sawQuery = r.URL.RawQuery
		sawHost = r.Host
		_, _ = w.Write([]byte(`{"BucketName":"ctk-demo","BucketId":"bid","NextMarker":"","DataSet":[{"FileName":"audit/2026-04-22.log","Hash":"h","MimeType":"application/json","Size":4096,"ModifyTime":1714694400}]}`))
	}))
	defer server.Close()

	// The endpoint format takes (bucket, region) — for the test we ignore
	// both and route every request to the httptest server.
	client := newTestFileClient(t, server.URL+"/%s%s")
	resp, err := client.PrefixFileList(context.Background(), "ctk-demo", "cn-bj", "", "", 100)
	if err != nil {
		t.Fatalf("PrefixFileList: %v", err)
	}
	if len(resp.DataSet) != 1 || resp.DataSet[0].Size != 4096 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if !strings.HasPrefix(sawAuth, "UCloud ucloudpubkey-test:") {
		t.Errorf("unexpected Authorization header: %q", sawAuth)
	}
	if sawDate == "" {
		t.Errorf("missing Date header")
	}
	if !strings.Contains(sawQuery, "list=") {
		t.Errorf("expected ?list, got %q", sawQuery)
	}
	if sawHost == "" {
		t.Errorf("expected Host header to be set")
	}
}

func TestPrefixFileListSurfacesUFileError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"RetCode":403,"ErrMsg":"signature mismatch"}`)
	}))
	defer server.Close()

	client := newTestFileClient(t, server.URL+"/%s%s")
	_, err := client.PrefixFileList(context.Background(), "ctk-demo", "cn-bj", "", "", 0)
	if err == nil {
		t.Fatalf("expected error on RetCode != 0")
	}
	if !strings.Contains(err.Error(), "signature mismatch") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSignRequestBuildsExpectedString(t *testing.T) {
	client := NewFileClient(
		auth.New("publickey", "privatekey", ""),
		WithFileClock(func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) }),
	)
	req, err := http.NewRequest(http.MethodGet, "https://example.com/?list", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	client.signRequest(req, "ctk-demo", "")
	if got := req.Header.Get("Authorization"); !strings.HasPrefix(got, "UCloud publickey:") {
		t.Fatalf("unexpected Authorization: %q", got)
	}
	if got := req.Header.Get("Date"); got == "" {
		t.Fatalf("expected Date header to be set")
	}
}

func TestListObjectsAggregatesPerBucket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"BucketName":"ctk-demo","DataSet":[{"FileName":"a","Size":10,"ModifyTime":1714694400},{"FileName":"b","Size":20,"ModifyTime":1714694401}]}`))
	}))
	defer server.Close()

	client := newTestFileClient(t, server.URL+"/%s%s")
	driver := &Driver{
		Credential: auth.New("ucloudpubkey-test", "ucloudprivkey-test", ""),
		FileClient: client,
		Region:     "cn-bj",
	}
	got, err := driver.ListObjects(context.Background(), map[string]string{"ctk-demo": "cn-bj"})
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].ObjectCount != 2 {
		t.Errorf("expected 2 objects, got %d", got[0].ObjectCount)
	}
	if len(got[0].Objects) != 2 || got[0].Objects[0].Key != "a" {
		t.Errorf("unexpected objects: %+v", got[0].Objects)
	}
}

func TestTotalObjectsSumsAcrossPages(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			_, _ = w.Write([]byte(`{"BucketName":"ctk-demo","NextMarker":"page-2","DataSet":[{"FileName":"a","Size":10},{"FileName":"b","Size":20}]}`))
		case 2:
			_, _ = w.Write([]byte(`{"BucketName":"ctk-demo","DataSet":[{"FileName":"c","Size":30}]}`))
		default:
			t.Fatalf("unexpected call: %d", calls)
		}
	}))
	defer server.Close()

	client := newTestFileClient(t, server.URL+"/%s%s")
	driver := &Driver{
		Credential: auth.New("ucloudpubkey-test", "ucloudprivkey-test", ""),
		FileClient: client,
		Region:     "cn-bj",
	}
	got, err := driver.TotalObjects(context.Background(), map[string]string{"ctk-demo": "cn-bj"})
	if err != nil {
		t.Fatalf("TotalObjects: %v", err)
	}
	if len(got) != 1 || got[0].ObjectCount != 3 {
		t.Fatalf("expected 3 objects, got %+v", got)
	}
	if !strings.Contains(got[0].Message, "60 bytes") {
		t.Errorf("expected total bytes in message, got %q", got[0].Message)
	}
}
