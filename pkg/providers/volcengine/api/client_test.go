package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
)

func TestClientListUsersSendsSignedGETRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.RawQuery; got != "Action=ListUsers&Limit=100&Offset=0&Version=2018-01-01" {
			t.Fatalf("unexpected query: %s", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded; charset=utf-8" {
			t.Fatalf("unexpected content type: %s", got)
		}
		if got := r.Header.Get(HeaderXDate); got != "20260419T120000Z" {
			t.Fatalf("unexpected x-date: %s", got)
		}
		if got := r.Header.Get(HeaderXContentSHA256); got != emptySHA256Hex {
			t.Fatalf("unexpected content hash: %s", got)
		}
		authz := r.Header.Get(HeaderAuthorization)
		if !strings.Contains(authz, "/cn-beijing/iam/request") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-1"},"Result":{"UserMetadata":[{"UserName":"alice","AccountId":1001,"CreateDate":"20260419T120000Z"}],"Total":1,"Limit":100,"Offset":0}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "")
	got, err := client.ListUsers(context.Background(), "cn-beijing", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(got.Result.UserMetadata) != 1 || got.Result.UserMetadata[0].UserName != "alice" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestClientQueryBalanceAcctSendsJSONBodyAndSessionToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.URL.RawQuery; got != "Action=QueryBalanceAcct&Version=2022-01-01" {
			t.Fatalf("unexpected query: %s", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("unexpected content type: %s", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("unexpected accept: %s", got)
		}
		if got := r.Header.Get(HeaderXSecurityToken); got != "token" {
			t.Fatalf("unexpected security token: %s", got)
		}
		if got := r.Header.Get(HeaderXContentSHA256); got != hashHex([]byte("{}")) {
			t.Fatalf("unexpected content hash: %s", got)
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-balance"},"Result":{"AvailableBalance":"123.45"}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "token")
	got, err := client.QueryBalanceAcct(context.Background(), "cn-beijing")
	if err != nil {
		t.Fatalf("QueryBalanceAcct() error = %v", err)
	}
	if got.Result.AvailableBalance != "123.45" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestClientReturnsAPIErrorWhenResponseMetadataCarriesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-error","Error":{"Code":"InvalidAccessKey.NotFound","Message":"bad ak"}}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "")
	_, err := client.GetLoginProfile(context.Background(), "cn-beijing", "alice")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "InvalidAccessKey.NotFound" || apiErr.Message != "bad ak" || apiErr.RequestID != "req-error" || apiErr.HTTPStatus != 200 {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func newTestClient(baseURL, token string) *Client {
	return NewClient(
		auth.New("AKID", "SECRET", token),
		WithBaseURL(baseURL),
		WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

const emptySHA256Hex = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func hashHex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
