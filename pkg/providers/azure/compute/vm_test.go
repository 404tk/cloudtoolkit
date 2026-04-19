package compute

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
)

func TestDriverGetResourceMapsVMs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/subscriptions/sub-1/resourceGroups":
			_, _ = w.Write([]byte(`{"value":[{"name":"rg-1"}]}`))
		case "/subscriptions/sub-2/resourceGroups":
			_, _ = w.Write([]byte(`{"value":[{"name":"rg-2"}]}`))
		case "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines":
			_, _ = w.Write([]byte(`{"value":[{"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-public","name":"vm-public","location":"eastasia","properties":{"provisioningState":"Succeeded","networkProfile":{"networkInterfaces":[{"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Network/networkInterfaces/nic-public"}]}}}]}`))
		case "/subscriptions/sub-2/resourceGroups/rg-2/providers/Microsoft.Compute/virtualMachines":
			_, _ = w.Write([]byte(`{"value":[{"id":"/subscriptions/sub-2/resourceGroups/rg-2/providers/Microsoft.Compute/virtualMachines/vm-private","name":"vm-private","location":"chinaeast2","properties":{"provisioningState":"Succeeded","networkProfile":{"networkInterfaces":[{"id":"/subscriptions/sub-2/resourceGroups/rg-2/providers/Microsoft.Network/networkInterfaces/nic-private"}]}}}]}`))
		case "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Network/networkInterfaces/nic-public":
			_, _ = w.Write([]byte(`{"id":"nic-public","properties":{"ipConfigurations":[{"name":"ipconfig1","properties":{"privateIPAddress":"10.0.0.4","publicIPAddress":{"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Network/publicIPAddresses/pip-1"}}}]}}`))
		case "/subscriptions/sub-2/resourceGroups/rg-2/providers/Microsoft.Network/networkInterfaces/nic-private":
			_, _ = w.Write([]byte(`{"id":"nic-private","properties":{"ipConfigurations":[{"name":"ipconfig1","properties":{"privateIPAddress":"10.1.0.5"}}]}}`))
		case "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Network/publicIPAddresses/pip-1":
			_, _ = w.Write([]byte(`{"id":"pip-1","properties":{"ipAddress":"20.30.40.50"}}`))
		default:
			t.Fatalf("unexpected path: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Transport = tokenRewriteTransport{base: httpClient.Transport, target: mustParseURL(t, server.URL)}
	ts := auth.NewTokenSource(auth.New("client", "secret", "tenant", "", auth.CloudPublic), httpClient)
	client := azapi.NewClient(ts, cloud.For(auth.CloudPublic), azapi.WithHTTPClient(httpClient), azapi.WithBaseURL(server.URL))

	driver := &Driver{
		Client:          client,
		SubscriptionIDs: []string{"sub-1", "sub-2"},
	}
	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected host count: %d", len(got))
	}
	if got[0].HostName != "vm-public" || got[0].PublicIPv4 != "20.30.40.50" || !got[0].Public || got[0].PrivateIpv4 != "10.0.0.4" {
		t.Fatalf("unexpected public host: %+v", got[0])
	}
	if got[1].HostName != "vm-private" || got[1].Public || got[1].PrivateIpv4 != "10.1.0.5" {
		t.Fatalf("unexpected private host: %+v", got[1])
	}
}

type tokenRewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (rt tokenRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "login.microsoftonline.com" {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = rt.target.Scheme
		clone.URL.Host = rt.target.Host
		clone.Host = rt.target.Host
		return rt.base.RoundTrip(clone)
	}
	return rt.base.RoundTrip(req)
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return u
}
