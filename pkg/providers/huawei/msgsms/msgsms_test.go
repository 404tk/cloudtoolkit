package msgsms

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

type noopRetryPolicy struct{}

func (noopRetryPolicy) Do(ctx context.Context, _ bool, fn func() (*http.Response, error)) (*http.Response, error) {
	return fn()
}

func newTestClient(t *testing.T, fn roundTripFunc) *api.Client {
	t.Helper()
	return api.NewClient(
		auth.New("AKID", "SECRET", "cn-north-4", false),
		api.WithHTTPClient(&http.Client{Transport: fn}),
		api.WithRetryPolicy(noopRetryPolicy{}),
		api.WithClock(func() time.Time { return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC) }),
	)
}

func jsonResponse(r *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}
}

func TestGetResourceMapsSignsAndTemplates(t *testing.T) {
	driver := &Driver{
		Cred:     auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:  []string{"cn-north-4"},
		DomainID: "d-1",
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/v3/projects":
				return jsonResponse(r, `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`), nil
			case "/v2/project-n4/msgsms/signatures":
				return jsonResponse(r, `{"results":[{"id":"row-s1","signature_id":"s1","signature_name":"ctk-prod","status":"REVIEW_PASSED","signature_type":"NOTIFY_TYPE"},{"id":"row-s2","signature_id":"s2","signature_name":"ctk-stage","status":"PENDING_REVIEW","signature_type":"VERIFY_CODE_TYPE"}],"total":2}`), nil
			case "/v2/project-n4/msgsms/templates":
				return jsonResponse(r, `{"results":[{"id":"row-t1","template_id":"t1","template_name":"OTP","template_content":"Code is {1}","status":"REVIEW_PASSED","template_type":"VERIFY_CODE_TYPE"}],"total":1}`), nil
			}
			t.Fatalf("unexpected path: %s", r.URL.Path)
			return nil, nil
		})),
	}
	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if len(res.Signs) != 2 || res.Signs[0].Name != "ctk-prod" || res.Signs[0].Status != "REVIEW_PASSED" {
		t.Errorf("signs mismatch: %+v", res.Signs)
	}
	if len(res.Templates) != 1 || res.Templates[0].Name != "OTP" || res.Templates[0].Content == "" {
		t.Errorf("templates mismatch: %+v", res.Templates)
	}
}

func TestGetResourcePropagatesSignsError(t *testing.T) {
	driver := &Driver{
		Cred:     auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:  []string{"cn-north-4"},
		DomainID: "d-1",
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/v3/projects" {
				return jsonResponse(r, `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`), nil
			}
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error_code":"SMS.0403","error_msg":"forbidden"}`)),
				Request:    r,
			}, nil
		})),
	}
	_, err := driver.GetResource(context.Background())
	if err == nil {
		t.Fatal("expected error when listing signs fails")
	}
	if !strings.Contains(err.Error(), "SMS.0403") {
		t.Errorf("expected SMS.0403 in err, got %v", err)
	}
}

func TestGetResourceHandlesEmptyResults(t *testing.T) {
	driver := &Driver{
		Cred:     auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:  []string{"cn-north-4"},
		DomainID: "d-1",
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/v3/projects" {
				return jsonResponse(r, `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`), nil
			}
			return jsonResponse(r, `{"results":[],"total":0}`), nil
		})),
	}
	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if len(res.Signs) != 0 || len(res.Templates) != 0 {
		t.Errorf("expected empty result, got %+v", res)
	}
}
