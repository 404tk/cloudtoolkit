package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestDecodeErrorFromBody(t *testing.T) {
	err := DecodeError(http.StatusForbidden, []byte(`{"requestId":"req-1","error":{"status":"HTTP_FORBIDDEN","code":40301,"message":"denied"}}`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.RequestID != "req-1" || apiErr.Code != 40301 || apiErr.Status != "HTTP_FORBIDDEN" || apiErr.Message != "denied" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
	if apiErr.IsAuthFailure() {
		t.Fatal("403 should not be treated as auth failure")
	}
}

func TestDecodeErrorLeaves404UsableForValidator(t *testing.T) {
	err := DecodeError(http.StatusNotFound, []byte(`{"requestId":"req-404","error":{"status":404,"code":404,"message":"not found"}}`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.IsAuthFailure() {
		t.Fatal("404 should not be treated as auth failure")
	}
	if !strings.Contains(apiErr.Error(), "code=404") || !strings.Contains(apiErr.Error(), "request_id=req-404") {
		t.Fatalf("unexpected error string: %s", apiErr.Error())
	}
	if apiErr.Status != "404" {
		t.Fatalf("unexpected status: %q", apiErr.Status)
	}
}

func TestDecodeErrorTreats401AsAuthFailure(t *testing.T) {
	err := DecodeError(http.StatusUnauthorized, []byte(`{"requestId":"req-401","error":{"status":"HTTP_UNAUTHORIZED","code":401,"message":"unauthorized"}}`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if !apiErr.IsAuthFailure() {
		t.Fatal("401 should be treated as auth failure")
	}
}

func TestDecodeErrorNilOnSuccess(t *testing.T) {
	if err := DecodeError(http.StatusOK, []byte(`{"requestId":"req-ok","result":{"value":1}}`)); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestIsInvalidRegion(t *testing.T) {
	err := &APIError{
		HTTPStatus: http.StatusBadRequest,
		Code:       http.StatusBadRequest,
		Message:    "Invalid RegionId 'cn-east-2'",
	}
	if !IsInvalidRegion(err) {
		t.Fatal("expected invalid region error to be detected")
	}
	if IsInvalidRegion(&APIError{HTTPStatus: http.StatusBadRequest, Code: http.StatusBadRequest, Message: "other bad request"}) {
		t.Fatal("unexpected invalid region match")
	}
}
