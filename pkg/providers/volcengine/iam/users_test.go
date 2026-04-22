package iam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
)

func TestDriverListUsersPaginatesAndHandlesLoginProfileFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		switch values.Get("Action") {
		case "ListUsers":
			offset := values.Get("Offset")
			switch offset {
			case "0":
				_, _ = w.Write(mustJSON(t, listUsersResponse(0, 100, 101)))
			case "100":
				_, _ = w.Write(mustJSON(t, listUsersResponse(100, 1, 101)))
			default:
				t.Fatalf("unexpected offset: %s", offset)
			}
		case "GetLoginProfile":
			switch values.Get("UserName") {
			case "user-000":
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-login"},"Result":{"LoginProfile":{"UserName":"user-000","LoginAllowed":true,"LastLoginDate":"20260419T120100Z"}}}`))
			case "user-100":
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-miss","Error":{"Code":"EntityNotExist.User","Message":"not found"}}}`))
			default:
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-zero"},"Result":{"LoginProfile":{"UserName":"other","LoginAllowed":false,"LastLoginDate":"19700101T000000Z"}}}`))
			}
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client: newTestClient(server.URL),
		Region: "all",
	}
	got, err := driver.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(got) != 101 {
		t.Fatalf("unexpected user count: %d", len(got))
	}
	if got[0].UserName != "user-000" || !got[0].EnableLogin || got[0].LastLogin == "" {
		t.Fatalf("unexpected first user: %+v", got[0])
	}
	last := got[len(got)-1]
	if last.UserName != "user-100" || last.EnableLogin || last.LastLogin != "" {
		t.Fatalf("unexpected last user: %+v", last)
	}
	if got[0].CreateTime != "2026-04-19 12:00:00 +0000 UTC" {
		t.Fatalf("unexpected create time: %s", got[0].CreateTime)
	}
}

func TestDriverGetProjectFormatsName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		if values.Get("Action") != "ListProjects" {
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-project"},"Result":{"Projects":[{"ProjectName":"demo","AccountID":1002003}],"Total":1}}`))
	}))
	defer server.Close()

	driver := &Driver{
		Client: newTestClient(server.URL),
		Region: "",
	}
	got, err := driver.GetProject(context.Background())
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if got != "demo(1002003)" {
		t.Fatalf("unexpected project: %s", got)
	}
}

func newTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func listUsersResponse(offset, count, total int) api.ListUsersResponse {
	resp := api.ListUsersResponse{}
	resp.Result.Total = int32(total)
	resp.Result.Limit = 100
	resp.Result.Offset = int32(offset)
	resp.Result.UserMetadata = make([]api.IAMUserMetadata, 0, count)
	for i := 0; i < count; i++ {
		index := offset + i
		resp.Result.UserMetadata = append(resp.Result.UserMetadata, api.IAMUserMetadata{
			UserName:   "user-" + leftPad3(index),
			AccountID:  int64(1000 + index),
			CreateDate: "20260419T120000Z",
		})
	}
	return resp
}

func leftPad3(v int) string {
	switch {
	case v < 10:
		return "00" + strconv.Itoa(v)
	case v < 100:
		return "0" + strconv.Itoa(v)
	default:
		return strconv.Itoa(v)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}
