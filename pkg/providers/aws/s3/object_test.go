package s3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestListObjectsPageUsesDefaultRegionAndMaxKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bucket-a" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("list-type"); got != "2" {
			t.Fatalf("unexpected list-type: %s", got)
		}
		if got := r.URL.Query().Get("max-keys"); got != "100" {
			t.Fatalf("unexpected max-keys: %s", got)
		}
		if got := signingRegionFromAuthorization(t, r.Header.Get("Authorization")); got != "ap-southeast-1" {
			t.Fatalf("unexpected signing region: %s", got)
		}
		_, _ = w.Write([]byte(`
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>alpha.txt</Key>
    <Size>12</Size>
  </Contents>
</ListBucketResult>`))
	}))
	defer server.Close()

	driver := &Driver{
		Client:        newS3DriverTestClient(server.URL),
		DefaultRegion: "ap-southeast-1",
	}
	got, err := driver.listObjectsPage(context.Background(), "bucket-a", "", "", 100)
	if err != nil {
		t.Fatalf("listObjectsPage() error = %v", err)
	}
	if len(got.Objects) != 1 || got.Objects[0].Key != "alpha.txt" || got.Objects[0].Size != 12 {
		t.Fatalf("unexpected objects: %+v", got.Objects)
	}
}

func TestCountBucketObjectsPaginatesContinuationToken(t *testing.T) {
	var (
		mu      sync.Mutex
		request int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bucket-a" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := signingRegionFromAuthorization(t, r.Header.Get("Authorization")); got != "eu-west-1" {
			t.Fatalf("unexpected signing region: %s", got)
		}
		mu.Lock()
		request++
		current := request
		mu.Unlock()
		switch current {
		case 1:
			if got := r.URL.Query().Get("continuation-token"); got != "" {
				t.Fatalf("unexpected first continuation token: %s", got)
			}
			_, _ = w.Write([]byte(`
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <IsTruncated>true</IsTruncated>
  <NextContinuationToken>page-2</NextContinuationToken>
  <Contents><Key>a</Key><Size>1</Size></Contents>
  <Contents><Key>b</Key><Size>2</Size></Contents>
</ListBucketResult>`))
		case 2:
			if got := r.URL.Query().Get("continuation-token"); got != "page-2" {
				t.Fatalf("unexpected second continuation token: %s", got)
			}
			_, _ = w.Write([]byte(`
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <IsTruncated>false</IsTruncated>
  <Contents><Key>c</Key><Size>3</Size></Contents>
</ListBucketResult>`))
		default:
			t.Fatalf("unexpected request count: %d", current)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client: newS3DriverTestClient(server.URL),
	}
	got, err := driver.countBucketObjects(context.Background(), "bucket-a", "eu-west-1", nil)
	if err != nil {
		t.Fatalf("countBucketObjects() error = %v", err)
	}
	if got != 3 {
		t.Fatalf("unexpected object count: %d", got)
	}
}
