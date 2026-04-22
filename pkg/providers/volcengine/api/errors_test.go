package api

import "testing"

func TestDecodeErrorResponseMetadata(t *testing.T) {
	err := DecodeError(403, []byte(`{"ResponseMetadata":{"RequestId":"req-1","Error":{"Code":"SignatureFailure","Message":"bad signature"}}}`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "SignatureFailure" || apiErr.Message != "bad signature" || apiErr.RequestID != "req-1" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func TestDecodeErrorFallback(t *testing.T) {
	err := DecodeError(500, []byte(`not-json`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.HTTPStatus != 500 || apiErr.Message != "decoded body: not-json" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func TestDecodeErrorIgnoresSuccessWithoutError(t *testing.T) {
	err := DecodeError(200, []byte(`{"ResponseMetadata":{"RequestId":"req-1"},"Result":{"ok":true}}`))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestIsAccessDenied(t *testing.T) {
	err := &APIError{Code: "AccessDenied", Message: "User is not authorized to perform: ecs:DescribeInstances"}
	if !IsAccessDenied(err) {
		t.Fatalf("expected access denied match")
	}
}
