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
		Cred:    auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions: []string{"cn-north-4"},
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/v1/sms/signs":
				return jsonResponse(r, `{"signs":[{"sign_id":"s1","sign_name":"ctk-prod","sign_status":"PASSED","sign_type":"app"},{"sign_id":"s2","sign_name":"ctk-stage","sign_status":"PENDING","sign_type":"website"}],"total_count":2}`), nil
			case "/v1/sms/templates":
				return jsonResponse(r, `{"templates":[{"template_id":"t1","template_name":"OTP","content":"Code is {1}","template_status":"PASSED","template_type":"verification"}],"total_count":1}`), nil
			}
			t.Fatalf("unexpected path: %s", r.URL.Path)
			return nil, nil
		})),
	}
	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if len(res.Signs) != 2 || res.Signs[0].Name != "ctk-prod" || res.Signs[0].Status != "PASSED" {
		t.Errorf("signs mismatch: %+v", res.Signs)
	}
	if len(res.Templates) != 1 || res.Templates[0].Name != "OTP" || res.Templates[0].Content == "" {
		t.Errorf("templates mismatch: %+v", res.Templates)
	}
}

func TestGetResourcePropagatesSignsError(t *testing.T) {
	driver := &Driver{
		Cred:    auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions: []string{"cn-north-4"},
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
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
		Cred:    auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions: []string{"cn-north-4"},
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(r, `{"signs":[],"templates":[],"total_count":0}`), nil
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
