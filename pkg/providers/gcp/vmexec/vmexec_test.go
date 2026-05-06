package vmexec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func newClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	httpClient := server.Client()
	transport, err := testutil.RewriteHostsTransport(httpClient.Transport, server.URL, "compute.googleapis.com")
	if err != nil {
		t.Fatalf("RewriteHostsTransport: %v", err)
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

func TestExecuteSetsStartupScriptAndResets(t *testing.T) {
	var sawSetMetadata, sawReset bool
	var setMetadataBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/instances/vm-1") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"name":"vm-1","zone":"us-central1-a","status":"RUNNING","metadata":{"fingerprint":"FP1","items":[{"key":"foo","value":"bar"}]}}`))
		case strings.HasSuffix(r.URL.Path, "/instances/vm-1/setMetadata"):
			sawSetMetadata = true
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			setMetadataBody = string(buf)
			_, _ = w.Write([]byte(`{"name":"op-set","status":"DONE"}`))
		case strings.HasSuffix(r.URL.Path, "/instances/vm-1/reset"):
			sawReset = true
			_, _ = w.Write([]byte(`{"name":"op-reset","status":"DONE"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newClient(t, server), Projects: []string{"proj-1"}}
	res, err := driver.Execute(context.Background(), "us-central1-a/vm-1", "echo hello")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !sawSetMetadata {
		t.Error("expected setMetadata call")
	}
	if !sawReset {
		t.Error("expected reset call")
	}
	if !strings.Contains(setMetadataBody, "startup-script") {
		t.Errorf("setMetadata body missing startup-script key: %s", setMetadataBody)
	}
	if !strings.Contains(setMetadataBody, `"fingerprint":"FP1"`) {
		t.Errorf("expected fingerprint passthrough, got: %s", setMetadataBody)
	}
	if !strings.Contains(setMetadataBody, `"key":"foo"`) {
		t.Errorf("expected existing metadata items preserved, got: %s", setMetadataBody)
	}
	if !strings.Contains(res.Output, "Reboot triggered") {
		t.Errorf("unexpected output: %q", res.Output)
	}
}

func TestExecuteRejectsEmptyCommand(t *testing.T) {
	driver := &Driver{Client: nil, Projects: []string{"proj-1"}}
	if _, err := driver.Execute(context.Background(), "us-central1-a/vm-1", "  "); err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestExecutePropagatesOperationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/instances/vm-1") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"name":"vm-1","metadata":{"fingerprint":"FP","items":[]}}`))
		case strings.HasSuffix(r.URL.Path, "/setMetadata"):
			_, _ = w.Write([]byte(`{"name":"op","status":"DONE","error":{"errors":[{"code":"PERMISSION_DENIED","message":"missing compute.instances.setMetadata"}]}}`))
		default:
			t.Fatalf("unexpected: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newClient(t, server), Projects: []string{"proj-1"}}
	_, err := driver.Execute(context.Background(), "us-central1-a/vm-1", "echo hi")
	if err == nil {
		t.Fatal("expected propagated operation error")
	}
	if !strings.Contains(err.Error(), "PERMISSION_DENIED") {
		t.Errorf("expected PERMISSION_DENIED in err, got %v", err)
	}
}

func TestMergeStartupScriptDeduplicates(t *testing.T) {
	existing := api.InstanceMetadata{
		Fingerprint: "FP",
		Items: []api.InstanceMetadataItem{
			{Key: "foo", Value: "bar"},
			{Key: "startup-script", Value: "old script"},
		},
	}
	merged := mergeStartupScript(existing, "new script")
	if merged.Fingerprint != "FP" {
		t.Errorf("expected fingerprint preserved, got %q", merged.Fingerprint)
	}
	if len(merged.Items) != 2 {
		t.Fatalf("expected 2 items after merge, got %d", len(merged.Items))
	}
	if merged.Items[1].Key != "startup-script" || merged.Items[1].Value != "new script" {
		t.Errorf("expected new startup-script, got %+v", merged.Items[1])
	}
}
