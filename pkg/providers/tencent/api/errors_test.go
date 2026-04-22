package api

import (
	"errors"
	"net/http"
	"testing"
)

func TestDecodeErrorReturnsAPIError(t *testing.T) {
	err := DecodeError(http.StatusOK, []byte(`{"Response":{"Error":{"Code":"AuthFailure","Message":"denied"},"RequestId":"req-1"}}`))
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.Code != "AuthFailure" || apiErr.RequestID != "req-1" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func TestDecodeErrorFallsBackToHTTPStatus(t *testing.T) {
	err := DecodeError(http.StatusBadGateway, []byte("upstream broke"))
	var httpErr *HTTPStatusError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPStatusError, got %v", err)
	}
	if httpErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected status code: %d", httpErr.StatusCode)
	}
}

func TestDecodeErrorIgnoresSuccessfulResponseWithoutErrorEnvelope(t *testing.T) {
	if err := DecodeError(http.StatusOK, []byte(`{"Response":{"RequestId":"req-1"}}`)); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestIsAccessDenied(t *testing.T) {
	err := &APIError{Code: "UnauthorizedOperation", Message: "You are not authorized to perform operation."}
	if !IsAccessDenied(err) {
		t.Fatal("expected access denied match")
	}
}
