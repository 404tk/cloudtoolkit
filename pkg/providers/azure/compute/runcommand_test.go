package compute

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

func TestRunCommandUsesSubscriptionFromARMIDAndPollsLRO(t *testing.T) {
	var sawRunCommand bool
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/subscriptions/sub-2/resourceGroups/rg-2/providers/Microsoft.Compute/virtualMachines/vm-2/runCommand":
			sawRunCommand = true
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			w.Header().Set("Azure-AsyncOperation", server.URL+"/operation/1")
			w.Header().Set("Location", server.URL+"/result/1")
			w.WriteHeader(http.StatusAccepted)
		case "/operation/1":
			_, _ = w.Write([]byte(`{"status":"Succeeded"}`))
		case "/result/1":
			_, _ = w.Write([]byte(`{"value":[{"message":"hello from vm\n"}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:          newRunCommandTestClient(t, server),
		SubscriptionIDs: []string{"sub-1"},
		LROMaxPolls:     2,
	}
	out, err := driver.RunCommand(context.Background(), "/subscriptions/sub-2/resourceGroups/rg-2/providers/Microsoft.Compute/virtualMachines/vm-2", "linux", "whoami")
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
	if !sawRunCommand {
		t.Fatal("expected runCommand request")
	}
	if !strings.Contains(out, "hello from vm") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRunCommandAcceptsSubscriptionResourceGroupVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/subscriptions/sub-2/resourceGroups/rg-2/providers/Microsoft.Compute/virtualMachines/vm-2/runCommand":
			var body azapi.RunCommandInput
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.CommandID != "RunPowerShellScript" {
				t.Fatalf("unexpected command id: %s", body.CommandID)
			}
			_, _ = w.Write([]byte(`{"value":[{"message":"powershell output"}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:          newRunCommandTestClient(t, server),
		SubscriptionIDs: []string{"sub-1"},
	}
	out, err := driver.RunCommand(context.Background(), "sub-2/rg-2/vm-2", "windows", "Write-Host ok")
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
	if out != "powershell output" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRunCommandTwoPartIDUsesDefaultSubscription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1/runCommand":
			_, _ = w.Write([]byte(`{"value":[{"message":"default subscription"}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:          newRunCommandTestClient(t, server),
		SubscriptionIDs: []string{"sub-1", "sub-2"},
	}
	out, err := driver.RunCommand(context.Background(), "rg-1/vm-1", "linux", "hostname")
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
	if out != "default subscription" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func newRunCommandTestClient(t *testing.T, server *httptest.Server) *azapi.Client {
	t.Helper()
	httpClient := server.Client()
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	httpClient.Transport = tokenRewriteTransport{base: httpClient.Transport, target: target}
	ts := auth.NewTokenSource(auth.New("client", "secret", "tenant", "", auth.CloudPublic), httpClient)
	return azapi.NewClient(ts, cloud.For(auth.CloudPublic), azapi.WithHTTPClient(httpClient), azapi.WithBaseURL(server.URL))
}
