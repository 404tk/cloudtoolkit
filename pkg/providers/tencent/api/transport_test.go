package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewTransportDefaults(t *testing.T) {
	transport := NewTransport()
	if transport.MaxIdleConns != defaultMaxIdleConns {
		t.Fatalf("unexpected MaxIdleConns: %d", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != defaultMaxIdleConnsPerHost {
		t.Fatalf("unexpected MaxIdleConnsPerHost: %d", transport.MaxIdleConnsPerHost)
	}
	if transport.DisableKeepAlives {
		t.Fatal("keep alive should remain enabled")
	}
}

func TestHTTPClientTimeoutCanBeAppliedWithSharedTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: NewTransport(),
		Timeout:   20 * time.Millisecond,
	}
	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
