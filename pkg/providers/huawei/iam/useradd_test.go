package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDriverAddUserPrintsLoginURL(t *testing.T) {
	var addedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v3/users":
			if got := r.Header.Get("X-Domain-Id"); got != "d-1" {
				t.Fatalf("unexpected create-user domain header: %s", got)
			}
			if body := readBody(t, r); !strings.Contains(body, `"domain_id":"d-1"`) {
				t.Fatalf("unexpected create-user body: %s", body)
			}
			_, _ = w.Write([]byte(`{"user":{"id":"u-1","domain_id":"d-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v3/groups":
			if got := r.Header.Get("X-Domain-Id"); got != "d-1" {
				t.Fatalf("unexpected list-groups domain header: %s", got)
			}
			_, _ = w.Write([]byte(`{"groups":[{"id":"g-admin","name":"admin"}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v3/groups/g-admin/users/u-1":
			if got := r.Header.Get("X-Domain-Id"); got != "d-1" {
				t.Fatalf("unexpected add-to-group domain header: %s", got)
			}
			addedPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v3/auth/domains":
			_, _ = w.Write([]byte(`{"domains":[{"id":"d-1","name":"example"}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	driver.Username = "ctk"
	driver.Password = "P@ss"
	driver.DomainID = "d-1"
	result, err := driver.AddUser()

	if err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	if addedPath != "/v3/groups/g-admin/users/u-1" {
		t.Fatalf("unexpected add-user path: %s", addedPath)
	}
	if result.Username != "ctk" {
		t.Fatalf("unexpected username: %s", result.Username)
	}
	if result.Password != "P@ss" {
		t.Fatalf("unexpected password: %s", result.Password)
	}
	if !strings.Contains(result.LoginURL, "https://auth.huaweicloud.com/authui/login?id=example") {
		t.Fatalf("unexpected login URL: %s", result.LoginURL)
	}
}

func TestDriverAddUserFallsBackToAllGroups(t *testing.T) {
	added := make(map[string]bool)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v3/users":
			if got := r.Header.Get("X-Domain-Id"); got != "d-1" {
				t.Fatalf("unexpected create-user domain header: %s", got)
			}
			_, _ = w.Write([]byte(`{"user":{"id":"u-1","domain_id":"d-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v3/groups":
			if got := r.Header.Get("X-Domain-Id"); got != "d-1" {
				t.Fatalf("unexpected list-groups domain header: %s", got)
			}
			_, _ = w.Write([]byte(`{"groups":[{"id":"g-1","name":"ops"},{"id":"g-2","name":"readonly"}]}`))
		case r.Method == http.MethodPut:
			if got := r.Header.Get("X-Domain-Id"); got != "d-1" {
				t.Fatalf("unexpected add-to-group domain header: %s", got)
			}
			added[r.URL.Path] = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v3/auth/domains":
			_, _ = w.Write([]byte(`{"domains":[{"id":"d-1","name":"example"}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	driver.Username = "ctk"
	driver.Password = "P@ss"
	driver.DomainID = "d-1"
	_, err := driver.AddUser()

	if err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	if len(added) != 2 || !added["/v3/groups/g-1/users/u-1"] || !added["/v3/groups/g-2/users/u-1"] {
		t.Fatalf("unexpected fallback add paths: %+v", added)
	}
}

func TestDriverGetDomainNameReturnsEmptyWhenMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"domains":[{"id":"d-2","name":"other"}]}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	if got := driver.getDomainName(context.Background(), "d-1"); got != "" {
		t.Fatalf("unexpected domain name: %q", got)
	}
}

func TestDriverCreateUserResolvesDomainIDWhenMissing(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v3/auth/domains":
			_, _ = w.Write([]byte(`{"domains":[{"id":"d-1","name":"example"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v3/users":
			if got := r.Header.Get("X-Domain-Id"); got != "d-1" {
				t.Fatalf("unexpected create-user domain header: %s", got)
			}
			if body := readBody(t, r); !strings.Contains(body, `"domain_id":"d-1"`) {
				t.Fatalf("unexpected create-user body: %s", body)
			}
			_, _ = w.Write([]byte(`{"user":{"id":"u-1","domain_id":"d-1"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	driver.Username = "ctk"
	driver.Password = "P@ss"

	userID, domainID, err := driver.createUser(context.Background())
	if err != nil {
		t.Fatalf("createUser() error = %v", err)
	}
	if userID != "u-1" || domainID != "d-1" || driver.DomainID != "d-1" {
		t.Fatalf("unexpected createUser result: userID=%q domainID=%q cached=%q", userID, domainID, driver.DomainID)
	}
	if len(requests) != 2 || requests[0] != "/v3/auth/domains" || requests[1] != "/v3/users" {
		t.Fatalf("unexpected request sequence: %+v", requests)
	}
}

func TestDriverCreateUserWarnsWhenMultipleDomainsVisible(t *testing.T) {
	stdout, _ := withLoggerBuffers(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v3/auth/domains":
			_, _ = w.Write([]byte(`{"domains":[{"id":"d-1","name":"primary"},{"id":"d-2","name":"secondary"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v3/users":
			if got := r.Header.Get("X-Domain-Id"); got != "d-1" {
				t.Fatalf("unexpected create-user domain header: %s", got)
			}
			_, _ = w.Write([]byte(`{"user":{"id":"u-1","domain_id":"d-1"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	driver.Username = "ctk"
	driver.Password = "P@ss"

	_, _, err := driver.createUser(context.Background())
	if err != nil {
		t.Fatalf("createUser() error = %v", err)
	}
	if !strings.Contains(stdout.String(), `Multiple domains visible (2); proceeding with first "d-1". Verify this matches your AK's domain.`) {
		t.Fatalf("unexpected warning logs: %s", stdout.String())
	}
}
