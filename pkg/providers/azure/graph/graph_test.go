package graph

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
)

func newStubTokenSource(token string) *auth.TokenSource {
	cred := auth.New("client", "secret", "tenant", "subscription", "")
	ts := auth.NewTokenSourceForScope(cred, &http.Client{}, "https://graph.microsoft.com/.default")
	// Inject a known token directly so tests don't hit a real auth endpoint.
	auth.SetCachedToken(ts, auth.Token{
		AccessToken: token,
		ExpiresAt:   time.Now().Add(time.Hour),
	})
	return ts
}

// graph_test.go intentionally exercises ListPasswordCredentials, AddPassword,
// and RemovePassword against an httptest server that mimics Microsoft Graph.

func TestListPasswordCredentialsResolvesByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1.0/applications/app-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"app-1","appId":"app-1","displayName":"demo","passwordCredentials":[{"keyId":"k1","displayName":"baseline","startDateTime":"2026-04-20T08:00:00Z","endDateTime":"2027-04-20T08:00:00Z"}]}`))
	}))
	defer server.Close()

	client := NewClient(newStubTokenSource("token"), server.Client(), server.URL)
	app, err := client.ListPasswordCredentials(context.Background(), "app-1")
	if err != nil {
		t.Fatalf("ListPasswordCredentials: %v", err)
	}
	if len(app.PasswordCredentials) != 1 || app.PasswordCredentials[0].KeyID != "k1" {
		t.Errorf("unexpected creds: %+v", app.PasswordCredentials)
	}
}

func TestAddPasswordReturnsSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1.0/applications/app-1":
			_, _ = w.Write([]byte(`{"id":"app-1","appId":"app-1","displayName":"demo","passwordCredentials":[]}`))
		case "/v1.0/applications/app-1/addPassword":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"keyId":"new1","displayName":"ctk-test","startDateTime":"2026-04-30T09:00:00Z","endDateTime":"2027-04-30T09:00:00Z","secretText":"sekret"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(newStubTokenSource("token"), server.Client(), server.URL)
	pc, err := client.AddPassword(context.Background(), "app-1", "ctk-test")
	if err != nil {
		t.Fatalf("AddPassword: %v", err)
	}
	if pc.KeyID != "new1" || pc.SecretText != "sekret" {
		t.Errorf("unexpected pc: %+v", pc)
	}
}

func TestRemovePasswordPropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1.0/applications/app-1":
			_, _ = w.Write([]byte(`{"id":"app-1","appId":"app-1","displayName":"demo","passwordCredentials":[]}`))
		case "/v1.0/applications/app-1/removePassword":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"Request_ResourceNotFound","message":"key not found"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(newStubTokenSource("token"), server.Client(), server.URL)
	err := client.RemovePassword(context.Background(), "app-1", "missing")
	if err == nil {
		t.Fatalf("expected error from RemovePassword")
	}
	if !strings.Contains(err.Error(), "Request_ResourceNotFound") {
		t.Errorf("expected Request_ResourceNotFound, got %v", err)
	}
}

func TestResolveApplicationByAppID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1.0/applications/00000000-1111-2222-3333-444444444444":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"Request_ResourceNotFound","message":"not by id"}}`))
		case "/v1.0/applications":
			if !strings.Contains(r.URL.RawQuery, "appId+eq+%2700000000-1111-2222-3333-444444444444%27") &&
				!strings.Contains(r.URL.RawQuery, "appId%20eq%20%2700000000-1111-2222-3333-444444444444%27") {
				t.Logf("filter raw query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"value":[{"id":"obj-9","appId":"00000000-1111-2222-3333-444444444444","displayName":"resolved","passwordCredentials":[]}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(newStubTokenSource("token"), server.Client(), server.URL)
	app, err := client.ListPasswordCredentials(context.Background(), "00000000-1111-2222-3333-444444444444")
	if err != nil {
		t.Fatalf("ListPasswordCredentials: %v", err)
	}
	if app.ID != "obj-9" {
		t.Errorf("expected resolved id obj-9, got %s", app.ID)
	}
}

func TestCreateAndDeleteUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1.0/users":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"id":"obj-1","accountEnabled":true,"displayName":"ctk","mailNickname":"ctk","userPrincipalName":"ctk@example.com"}`))
		case "/v1.0/users/ctk@example.com":
			if r.Method != http.MethodDelete {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(newStubTokenSource("token"), server.Client(), server.URL)
	user, err := client.CreateUser(context.Background(), "ctk", "ctk@example.com", "Pwd!2026")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.ID != "obj-1" {
		t.Errorf("unexpected id: %s", user.ID)
	}
	if err := client.DeleteUser(context.Background(), "ctk@example.com"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
}

func TestListUsersPaginatesAndMapsFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1.0/users" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		rawQuery := r.URL.RawQuery
		switch {
		case strings.Contains(rawQuery, "skiptoken=page-2"):
			_, _ = w.Write([]byte(`{"value":[{"id":"obj-2","accountEnabled":false,"displayName":"disabled","userPrincipalName":"disabled@example.com"}]}`))
		case strings.Contains(rawQuery, "signInActivity"):
			_, _ = w.Write([]byte(`{"value":[{"id":"obj-1","accountEnabled":true,"displayName":"ctk","userPrincipalName":"ctk@example.com","createdDateTime":"2026-05-01T00:00:00Z","signInActivity":{"lastSignInDateTime":"2026-05-02T00:00:00Z"}}],"@odata.nextLink":"https://graph.microsoft.com/v1.0/users?$skiptoken=page-2"}`))
		default:
			t.Fatalf("unexpected raw query: %s", rawQuery)
		}
	}))
	defer server.Close()

	client := NewClient(newStubTokenSource("token"), server.Client(), server.URL)
	users, err := client.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].UserPrincipalName != "ctk@example.com" ||
		users[0].SignInActivity == nil ||
		users[0].SignInActivity.LastSignInDateTime == "" {
		t.Fatalf("unexpected first user: %+v", users[0])
	}
	if users[1].AccountEnabled {
		t.Fatalf("expected second user disabled: %+v", users[1])
	}
}
