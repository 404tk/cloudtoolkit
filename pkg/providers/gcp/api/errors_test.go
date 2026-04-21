package api

import (
	"net/http"
	"testing"
)

func TestDecodeError(t *testing.T) {
	err := DecodeError(http.StatusForbidden, []byte(`{"error":{"code":403,"message":"Permission denied on resource project ctk.","status":"PERMISSION_DENIED","errors":[{"message":"Permission denied on resource project ctk.","domain":"global","reason":"forbidden"}]}}`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != 403 || apiErr.Status != "PERMISSION_DENIED" || apiErr.Reason != "forbidden" || apiErr.Domain != "global" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
	if got := apiErr.Error(); got != "gcp api error: status=403 code=PERMISSION_DENIED reason=forbidden: Permission denied on resource project ctk." {
		t.Fatalf("unexpected error string: %s", got)
	}
	if !IsAuthFailure(err) {
		t.Fatal("expected auth failure")
	}
}

func TestDecodeErrorFallback(t *testing.T) {
	err := DecodeError(http.StatusInternalServerError, []byte("<html>boom</html>"))
	if err == nil || err.Error() != "gcp api error: status=500 body=<html>boom</html>" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPredicates(t *testing.T) {
	if !IsNotFound(&APIError{StatusCode: http.StatusNotFound}) {
		t.Fatal("expected not found")
	}
	if !IsNotFound(&APIError{Status: "NOT_FOUND"}) {
		t.Fatal("expected not found by status")
	}
	if !IsRateLimited(&APIError{StatusCode: http.StatusTooManyRequests}) {
		t.Fatal("expected rate limited")
	}
	if !IsRateLimited(&APIError{Status: "RESOURCE_EXHAUSTED"}) {
		t.Fatal("expected rate limited by status")
	}
}
