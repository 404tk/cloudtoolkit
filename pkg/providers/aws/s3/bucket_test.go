package s3

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

func TestDriverGetBucketsPrefersLocationThenHintThenDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`
<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Buckets>
    <Bucket>
      <Name>alpha</Name>
      <BucketRegion>ap-east-1</BucketRegion>
    </Bucket>
    <Bucket>
      <Name>beta</Name>
      <BucketRegion>ap-southeast-1</BucketRegion>
    </Bucket>
    <Bucket>
      <Name>gamma</Name>
    </Bucket>
  </Buckets>
</ListAllMyBucketsResult>`))
		case "/alpha":
			if _, ok := r.URL.Query()["location"]; !ok {
				t.Fatalf("missing location query for alpha")
			}
			_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-west-2</LocationConstraint>`))
		case "/beta":
			_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`))
		case "/gamma":
			_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:        newS3DriverTestClient(server.URL),
		DefaultRegion: "us-east-1",
	}
	got, err := driver.GetBuckets(context.Background())
	if err != nil {
		t.Fatalf("GetBuckets() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("unexpected bucket count: %d", len(got))
	}
	if got[0].BucketName != "alpha" || got[0].Region != "us-west-2" {
		t.Fatalf("unexpected alpha bucket: %+v", got[0])
	}
	if got[1].BucketName != "beta" || got[1].Region != "ap-southeast-1" {
		t.Fatalf("unexpected beta bucket: %+v", got[1])
	}
	if got[2].BucketName != "gamma" || got[2].Region != "us-east-1" {
		t.Fatalf("unexpected gamma bucket: %+v", got[2])
	}
}

func newS3DriverTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC) }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func signingRegionFromAuthorization(t *testing.T, authorization string) string {
	t.Helper()
	const prefix = "Credential="
	start := strings.Index(authorization, prefix)
	if start < 0 {
		t.Fatalf("missing credential scope: %s", authorization)
	}
	scope := authorization[start+len(prefix):]
	if end := strings.Index(scope, ","); end >= 0 {
		scope = scope[:end]
	}
	parts := strings.Split(scope, "/")
	if len(parts) < 5 {
		t.Fatalf("invalid credential scope: %s", authorization)
	}
	return parts[2]
}
