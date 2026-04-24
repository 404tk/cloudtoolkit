package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

func TestClientDoMatchesCapturedFormBody(t *testing.T) {
	for _, fx := range loadSignerFixtures(t) {
		fx := fx
		t.Run(fx.Name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("unexpected method: %s", r.Method)
				}
				if got := r.Header.Get("Content-Type"); got != fx.Expected.ContentType {
					t.Fatalf("unexpected content type: %s", got)
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				if got := string(body); got != fx.Expected.FormBody {
					t.Fatalf("unexpected body:\n got: %s\nwant: %s", got, fx.Expected.FormBody)
				}

				_, _ = w.Write([]byte(`{"RetCode":0}`))
			}))
			defer server.Close()

			client := NewClient(
				ucloudauth.New(fx.Input.AccessKey, fx.Input.SecretKey, fx.Input.SecurityToken),
				WithBaseURL(server.URL),
			)

			err := client.Do(context.Background(), Request{
				Action:    fx.Input.Action,
				Region:    fx.Input.Region,
				ProjectID: fx.Input.ProjectID,
				Params:    fx.Input.Params,
			}, nil)
			if err != nil {
				t.Fatalf("Do() error = %v", err)
			}
		})
	}
}

func TestClientDoAcceptsStringRetCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"RetCode":"0","DataSet":[{"UserName":"alice","UserEmail":"alice@example.com","UserId":7}]}`))
	}))
	defer server.Close()

	client := NewClient(
		ucloudauth.New("foo", "bar", ""),
		WithBaseURL(server.URL),
	)

	var resp GetUserInfoResponse
	err := client.Do(context.Background(), Request{Action: "GetUserInfo"}, &resp)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if len(resp.DataSet) != 1 || resp.DataSet[0].UserName != "alice" || resp.DataSet[0].UserID != 7 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClientDoReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"RetCode":230,"Message":"Params [ClassType] not valid"}`))
	}))
	defer server.Close()

	client := NewClient(
		ucloudauth.New("foo", "bar", ""),
		WithBaseURL(server.URL),
	)

	err := client.Do(context.Background(), Request{Action: "DescribeUDBInstance"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != 230 || apiErr.Message != "Params [ClassType] not valid" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func TestClientDoHonorsCanceledContext(t *testing.T) {
	client := NewClient(
		ucloudauth.New("foo", "bar", ""),
		WithBaseURL("http://127.0.0.1:1"),
		WithHTTPClient(&http.Client{Timeout: time.Second}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Do(ctx, Request{Action: "GetUserInfo"}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
