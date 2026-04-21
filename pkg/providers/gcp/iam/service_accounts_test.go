package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func TestListUsersSkipsBlankDisplayNameAndHandlesPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/v1/projects/proj-1/serviceAccounts":
			switch r.URL.Query().Get("pageToken") {
			case "":
				if got := r.URL.Query().Get("pageSize"); got != "100" {
					t.Fatalf("unexpected pageSize: %s", got)
				}
				_, _ = w.Write([]byte(`{"accounts":[{"displayName":"sa-one","uniqueId":"1"},{"displayName":"","uniqueId":"skip"}],"nextPageToken":"page-2"}`))
			case "page-2":
				_, _ = w.Write([]byte(`{"accounts":[{"displayName":"sa-two","uniqueId":"2"}]}`))
			default:
				t.Fatalf("unexpected pageToken: %s", r.URL.Query().Get("pageToken"))
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Projects: []string{"proj-1"},
		Client:   newTestClient(t, server),
	}
	users, err := driver.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("unexpected user count: %d", len(users))
	}
	if users[0].UserName != "sa-one" || users[0].UserId != "1" {
		t.Fatalf("unexpected first user: %+v", users[0])
	}
	if users[1].UserName != "sa-two" || users[1].UserId != "2" {
		t.Fatalf("unexpected second user: %+v", users[1])
	}
}

func newTestClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	httpClient := server.Client()
	transport, err := testutil.RewriteHostsTransport(httpClient.Transport, server.URL, "iam.googleapis.com")
	if err != nil {
		t.Fatalf("RewriteHostsTransport() error = %v", err)
	}
	httpClient.Transport = transport

	ts := auth.NewTokenSource(auth.Credential{
		Type:          "service_account",
		ProjectID:     "proj-1",
		PrivateKeyID:  "kid-1",
		PrivateKeyPEM: testutil.PKCS8PrivateKeyPEM,
		ClientEmail:   "demo@example.com",
		TokenURI:      server.URL + "/token",
		Scopes:        []string{auth.DefaultScope},
	}, httpClient)
	return api.NewClient(ts, api.WithHTTPClient(httpClient))
}
