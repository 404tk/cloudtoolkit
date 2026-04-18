package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func TestGetCallerIdentitySendsSignedQueryRequestAndParsesXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const expectedBody = "Action=GetCallerIdentity&Version=2011-06-15"
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded; charset=utf-8" {
			t.Fatalf("unexpected content type: %s", got)
		}
		if got := r.Header.Get("X-Amz-Date"); got != "20260418T120000Z" {
			t.Fatalf("unexpected x-amz-date: %s", got)
		}
		if got := r.Header.Get("Content-Length"); got != "43" {
			t.Fatalf("unexpected content length: %s", got)
		}
		if got := readBody(t, r); got != expectedBody {
			t.Fatalf("unexpected body: %s", got)
		}
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "AWS4-HMAC-SHA256 Credential=AKID/20260418/us-east-1/sts/aws4_request, SignedHeaders=content-length;content-type;host;x-amz-date, Signature=") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`
<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <GetCallerIdentityResult>
    <Arn>arn:aws:iam::123456789012:user/demo</Arn>
    <UserId>AIDAEXAMPLE</UserId>
    <Account>123456789012</Account>
  </GetCallerIdentityResult>
  <ResponseMetadata>
    <RequestId>req-123</RequestId>
  </ResponseMetadata>
</GetCallerIdentityResponse>`))
	}))
	defer server.Close()

	client := NewClient(
		auth.New("AKID", "SECRET", ""),
		WithBaseURL(server.URL),
		WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)

	got, err := client.GetCallerIdentity(context.Background(), "us-east-1")
	if err != nil {
		t.Fatalf("GetCallerIdentity() error = %v", err)
	}
	if got.Arn != "arn:aws:iam::123456789012:user/demo" || got.Account != "123456789012" || got.UserID != "AIDAEXAMPLE" || got.RequestID != "req-123" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetCallerIdentityUsesRequestedRegionalSigningScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "AWS4-HMAC-SHA256 Credential=AKID/20260418/ap-southeast-1/sts/aws4_request, SignedHeaders=content-length;content-type;host;x-amz-date, Signature=") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`
<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <GetCallerIdentityResult>
    <Arn>arn:aws:iam::123456789012:user/demo</Arn>
    <UserId>AIDAREGIONAL</UserId>
    <Account>123456789012</Account>
  </GetCallerIdentityResult>
  <ResponseMetadata>
    <RequestId>req-regional</RequestId>
  </ResponseMetadata>
</GetCallerIdentityResponse>`))
	}))
	defer server.Close()

	client := NewClient(
		auth.New("AKID", "SECRET", ""),
		WithBaseURL(server.URL),
		WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)

	got, err := client.GetCallerIdentity(context.Background(), "ap-southeast-1")
	if err != nil {
		t.Fatalf("GetCallerIdentity() error = %v", err)
	}
	if got.RequestID != "req-regional" || got.UserID != "AIDAREGIONAL" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetCallerIdentityIncludesSessionToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Amz-Security-Token"); got != "token" {
			t.Fatalf("unexpected session token: %s", got)
		}
		authz := r.Header.Get("Authorization")
		if !strings.Contains(authz, "SignedHeaders=content-length;content-type;host;x-amz-date;x-amz-security-token") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`
<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <GetCallerIdentityResult>
    <Arn>arn:aws-cn:iam::210987654321:user/demo</Arn>
    <UserId>AIDATOKEN</UserId>
    <Account>210987654321</Account>
  </GetCallerIdentityResult>
  <ResponseMetadata>
    <RequestId>req-token</RequestId>
  </ResponseMetadata>
</GetCallerIdentityResponse>`))
	}))
	defer server.Close()

	client := NewClient(
		auth.New("AKID", "SECRET", "token"),
		WithBaseURL(server.URL),
		WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)

	got, err := client.GetCallerIdentity(context.Background(), "cn-northwest-1")
	if err != nil {
		t.Fatalf("GetCallerIdentity() error = %v", err)
	}
	if got.RequestID != "req-token" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetCallerIdentityReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`
<ErrorResponse>
  <Error>
    <Type>Sender</Type>
    <Code>InvalidClientTokenId</Code>
    <Message>The security token included in the request is invalid.</Message>
  </Error>
  <RequestId>req-error</RequestId>
</ErrorResponse>`))
	}))
	defer server.Close()

	client := NewClient(
		auth.New("AKID", "SECRET", ""),
		WithBaseURL(server.URL),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)

	_, err := client.GetCallerIdentity(context.Background(), "us-east-1")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "InvalidClientTokenId" || apiErr.RequestID != "req-error" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}
