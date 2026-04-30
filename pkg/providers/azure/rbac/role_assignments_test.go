package rbac

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
)

const (
	testSubscription = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	testReaderGUID   = "acdd72a7-3385-48ef-bd42-f606fba81ae7"
	testPrincipal    = "11111111-2222-3333-4444-555555555555"
)

func TestList(t *testing.T) {
	server := newServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/oauth2/v2.0/token"):
			writeJSON(t, w, http.StatusOK, map[string]any{
				"access_token": "tok", "token_type": "Bearer", "expires_in": 3600,
			})
		case strings.HasSuffix(r.URL.Path, "/providers/Microsoft.Authorization/roleAssignments") && r.Method == http.MethodGet:
			if got := r.URL.Query().Get("api-version"); got != azapi.AuthorizationAPIVersion {
				t.Errorf("unexpected api-version: %s", got)
			}
			writeJSON(t, w, http.StatusOK, azapi.ListRoleAssignmentsResponse{
				Value: []azapi.RoleAssignment{{
					Name: "assign-1",
					Properties: azapi.RoleAssignmentProperties{
						RoleDefinitionID: "/subscriptions/" + testSubscription + "/providers/Microsoft.Authorization/roleDefinitions/" + testReaderGUID,
						PrincipalID:      testPrincipal,
					},
				}},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))

	driver := &Driver{Client: server.client, SubscriptionIDs: []string{testSubscription}}
	got, err := driver.List(context.Background(), driver.DefaultScope(), "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].Name != "assign-1" {
		t.Fatalf("unexpected assignments: %+v", got)
	}
}

func TestCreateResolvesRoleNameAndPUTs(t *testing.T) {
	var sawRoleDefs bool
	var sawPut bool
	var putBody []byte
	server := newServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/oauth2/v2.0/token"):
			writeJSON(t, w, http.StatusOK, map[string]any{
				"access_token": "tok", "token_type": "Bearer", "expires_in": 3600,
			})
		case strings.HasSuffix(r.URL.Path, "/providers/Microsoft.Authorization/roleDefinitions") && r.Method == http.MethodGet:
			sawRoleDefs = true
			if filter := r.URL.Query().Get("$filter"); !strings.Contains(filter, "roleName eq 'Reader'") {
				t.Errorf("unexpected filter: %s", filter)
			}
			writeJSON(t, w, http.StatusOK, azapi.ListRoleDefinitionsResponse{
				Value: []azapi.RoleDefinition{{
					ID:         "/subscriptions/" + testSubscription + "/providers/Microsoft.Authorization/roleDefinitions/" + testReaderGUID,
					Name:       testReaderGUID,
					Properties: azapi.RoleDefinitionProperties{RoleName: "Reader"},
				}},
			})
		case strings.Contains(r.URL.Path, "/providers/Microsoft.Authorization/roleAssignments/") && r.Method == http.MethodPut:
			sawPut = true
			body := readAll(t, r)
			putBody = body
			var payload azapi.CreateRoleAssignmentRequest
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("unmarshal put body: %v", err)
			}
			if payload.Properties.PrincipalID != testPrincipal {
				t.Errorf("unexpected principalId: %s", payload.Properties.PrincipalID)
			}
			if !strings.HasSuffix(payload.Properties.RoleDefinitionID, testReaderGUID) {
				t.Errorf("unexpected roleDefinitionId: %s", payload.Properties.RoleDefinitionID)
			}
			parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
			assignmentName := parts[len(parts)-1]
			writeJSON(t, w, http.StatusCreated, azapi.RoleAssignment{
				Name:       assignmentName,
				Properties: payload.Properties,
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))

	driver := &Driver{Client: server.client, SubscriptionIDs: []string{testSubscription}}
	created, err := driver.Create(context.Background(), driver.DefaultScope(), testPrincipal, "Reader")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !sawRoleDefs {
		t.Error("expected roleDefinitions lookup")
	}
	if !sawPut {
		t.Error("expected PUT roleAssignment")
	}
	if created.Name == "" {
		t.Errorf("expected non-empty assignment name; body was %s", string(putBody))
	}
}

func TestDeleteByPrincipalAndRole(t *testing.T) {
	deleteCalled := false
	server := newServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/oauth2/v2.0/token"):
			writeJSON(t, w, http.StatusOK, map[string]any{
				"access_token": "tok", "token_type": "Bearer", "expires_in": 3600,
			})
		case strings.HasSuffix(r.URL.Path, "/providers/Microsoft.Authorization/roleDefinitions"):
			writeJSON(t, w, http.StatusOK, azapi.ListRoleDefinitionsResponse{
				Value: []azapi.RoleDefinition{{
					ID:         "/subscriptions/" + testSubscription + "/providers/Microsoft.Authorization/roleDefinitions/" + testReaderGUID,
					Name:       testReaderGUID,
					Properties: azapi.RoleDefinitionProperties{RoleName: "Reader"},
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/providers/Microsoft.Authorization/roleAssignments") && r.Method == http.MethodGet:
			writeJSON(t, w, http.StatusOK, azapi.ListRoleAssignmentsResponse{
				Value: []azapi.RoleAssignment{{
					Name: "assign-target",
					Properties: azapi.RoleAssignmentProperties{
						RoleDefinitionID: "/subscriptions/" + testSubscription + "/providers/Microsoft.Authorization/roleDefinitions/" + testReaderGUID,
						PrincipalID:      testPrincipal,
					},
				}},
			})
		case strings.HasSuffix(r.URL.Path, "/providers/Microsoft.Authorization/roleAssignments/assign-target") && r.Method == http.MethodDelete:
			deleteCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))

	driver := &Driver{Client: server.client, SubscriptionIDs: []string{testSubscription}}
	got, err := driver.Delete(context.Background(), driver.DefaultScope(), "", testPrincipal, "Reader")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got != "assign-target" {
		t.Errorf("expected assign-target; got %s", got)
	}
	if !deleteCalled {
		t.Error("expected DELETE call")
	}
}

func TestNormalizeScope(t *testing.T) {
	cases := map[string]string{
		"":                                  "",
		"/subscriptions/abc":                "/subscriptions/abc",
		"subscriptions/abc":                 "/subscriptions/abc",
		"/subscriptions/abc/":               "/subscriptions/abc",
		"  /subscriptions/abc  ":            "/subscriptions/abc",
	}
	for in, want := range cases {
		if got := normalizeScope(in); got != want {
			t.Errorf("normalizeScope(%q) = %q; want %q", in, got, want)
		}
	}
}

// --- helpers ---

type rbacTestServer struct {
	*httptest.Server
	client *azapi.Client
}

func newServer(t *testing.T, handler http.Handler) *rbacTestServer {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	httpClient := srv.Client()
	httpClient.Transport = rewriteTokenHost(t, httpClient.Transport, srv.URL)
	ts := auth.NewTokenSource(auth.New("client", "secret", "tenant", "", auth.CloudPublic), httpClient)
	apiClient := azapi.NewClient(ts, cloud.For(auth.CloudPublic), azapi.WithHTTPClient(httpClient), azapi.WithBaseURL(srv.URL))
	return &rbacTestServer{Server: srv, client: apiClient}
}

type tokenRewriteTransport struct {
	t      *testing.T
	base   http.RoundTripper
	target *url.URL
}

func rewriteTokenHost(t *testing.T, base http.RoundTripper, rawTarget string) http.RoundTripper {
	t.Helper()
	if base == nil {
		base = http.DefaultTransport
	}
	target, err := url.Parse(rawTarget)
	if err != nil {
		t.Fatalf("parse target url: %v", err)
	}
	return tokenRewriteTransport{t: t, base: base, target: target}
}

func (rt tokenRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "login.microsoftonline.com" {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = rt.target.Scheme
		clone.URL.Host = rt.target.Host
		return rt.base.RoundTrip(clone)
	}
	return rt.base.RoundTrip(req)
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func readAll(t *testing.T, r *http.Request) []byte {
	t.Helper()
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	buf := make([]byte, r.ContentLength)
	if r.ContentLength > 0 {
		if _, err := r.Body.Read(buf); err != nil && err.Error() != "EOF" {
			t.Fatalf("read request body: %v", err)
		}
	}
	return buf
}
