package api

import (
	"net/http"
	"testing"
)

func TestDecodeError(t *testing.T) {
	err := DecodeError(http.StatusForbidden, []byte(`{"error":{"code":"AuthorizationFailed","message":"denied"}}`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "AuthorizationFailed" || apiErr.Message != "denied" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
	if !IsAuthFailure(err) {
		t.Fatal("expected auth failure")
	}
}

func TestIsNotFound(t *testing.T) {
	err := &APIError{StatusCode: http.StatusNotFound, Code: "ResourceNotFound"}
	if !IsNotFound(err) {
		t.Fatal("expected not found")
	}
}
