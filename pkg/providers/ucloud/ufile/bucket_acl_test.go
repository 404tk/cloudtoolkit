package ufile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ucloudapi "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

func newACLDriver(t *testing.T, baseURL, region string) *Driver {
	t.Helper()
	credential := auth.New("ucloudpubkey-test", "ucloudprivkey-test", "")
	client := ucloudapi.NewClient(credential,
		ucloudapi.WithBaseURL(baseURL),
		ucloudapi.WithRetryPolicy(ucloudapi.RetryPolicy{MaxAttempts: 1}),
	)
	return &Driver{
		Credential: credential,
		Client:     client,
		Region:     region,
	}
}

func TestAuditBucketACLMapsType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("Action") != "DescribeBucket" {
			t.Fatalf("unexpected action: %s", r.Form.Get("Action"))
		}
		_, _ = w.Write([]byte(`{"Action":"DescribeBucketResponse","RetCode":0,"DataSet":[{"BucketName":"a","Region":"cn-bj","Type":"private"},{"BucketName":"b","Region":"cn-sh","Type":"public"}]}`))
	}))
	defer server.Close()

	driver := newACLDriver(t, server.URL, "")
	got, err := driver.AuditBucketACL(context.Background(), "")
	if err != nil {
		t.Fatalf("AuditBucketACL: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].Container != "a" || got[0].Level != UFileTypePrivate {
		t.Errorf("unexpected first entry: %+v", got[0])
	}
	if got[1].Level != UFileTypePublic {
		t.Errorf("unexpected second level: %s", got[1].Level)
	}
}

func TestAuditBucketACLFiltersSpecificBucketLocally(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		_, _ = w.Write([]byte(`{"Action":"DescribeBucketResponse","RetCode":0,"DataSet":[{"BucketName":"a","Region":"cn-bj","Type":"private"},{"BucketName":"b","Region":"cn-sh","Type":"public"}]}`))
	}))
	defer server.Close()

	driver := newACLDriver(t, server.URL, "")
	got, err := driver.AuditBucketACL(context.Background(), "b")
	if err != nil {
		t.Fatalf("AuditBucketACL: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0].Container != "b" || got[0].Level != UFileTypePublic {
		t.Fatalf("unexpected filtered entry: %+v", got[0])
	}
}

func TestExposeBucketDefaultsToPublic(t *testing.T) {
	var sawAction string
	var sawType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		sawAction = r.Form.Get("Action")
		sawType = r.Form.Get("Type")
		_, _ = w.Write([]byte(`{"Action":"UpdateBucketResponse","RetCode":0,"BucketName":"a"}`))
	}))
	defer server.Close()

	driver := newACLDriver(t, server.URL, "")
	applied, err := driver.ExposeBucket(context.Background(), "a", "")
	if err != nil {
		t.Fatalf("ExposeBucket: %v", err)
	}
	if applied != UFileTypePublic {
		t.Errorf("expected public, got %q", applied)
	}
	if sawAction != "UpdateBucket" || sawType != UFileTypePublic {
		t.Errorf("unexpected request action=%s type=%s", sawAction, sawType)
	}
}

func TestUnexposeBucketSendsPrivate(t *testing.T) {
	var sawType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		sawType = r.Form.Get("Type")
		_, _ = w.Write([]byte(`{"Action":"UpdateBucketResponse","RetCode":0}`))
	}))
	defer server.Close()

	driver := newACLDriver(t, server.URL, "")
	if err := driver.UnexposeBucket(context.Background(), "a"); err != nil {
		t.Fatalf("UnexposeBucket: %v", err)
	}
	if sawType != UFileTypePrivate {
		t.Errorf("expected Type=private, got %q", sawType)
	}
}

func TestExposeBucketSurfacesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Action":"UpdateBucketResponse","RetCode":54023,"Message":"bucket not found"}`))
	}))
	defer server.Close()

	driver := newACLDriver(t, server.URL, "")
	_, err := driver.ExposeBucket(context.Background(), "missing", "")
	if err == nil {
		t.Fatalf("expected error for non-zero RetCode")
	}
	if !strings.Contains(err.Error(), "bucket not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNormalizeUFileType(t *testing.T) {
	cases := map[string]string{
		"":            UFileTypePrivate,
		"private":     UFileTypePrivate,
		"public":      UFileTypePublic,
		"PUBLIC":      UFileTypePublic,
		"public-read": UFileTypePublic,
		"limited":     UFileTypeLimited,
		"trigger":     UFileTypeLimited,
		"unknown":     "unknown",
	}
	for in, want := range cases {
		if got := normalizeUFileType(in); got != want {
			t.Errorf("normalizeUFileType(%q) = %q, want %q", in, got, want)
		}
	}
}
