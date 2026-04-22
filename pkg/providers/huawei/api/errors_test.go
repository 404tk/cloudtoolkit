package api

import "testing"

func TestDecodeErrorLegacy(t *testing.T) {
	err := DecodeError(400, []byte(`{"error_code":"IAM.0001","error_msg":"bad request"}`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "IAM.0001" || apiErr.Message != "bad request" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func TestDecodeErrorKeystone(t *testing.T) {
	err := DecodeError(403, []byte(`{"error":{"code":"IAM.0002","message":"forbidden","title":"Forbidden"}}`))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "IAM.0002" || apiErr.Message != "forbidden" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func TestDecodeErrorFallback(t *testing.T) {
	err := DecodeError(500, nil)
	if err == nil {
		t.Fatal("expected fallback error")
	}
	if got := err.Error(); got == "" || got == "Internal Server Error" {
		t.Fatalf("unexpected fallback error: %q", got)
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(&APIError{StatusCode: 404}) {
		t.Fatal("expected 404 to be treated as not found")
	}
	if !IsNotFound(&APIError{Code: "IAM.ItemNotExist"}) {
		t.Fatal("expected ItemNotExist suffix to match")
	}
	if !IsNotFound(&APIError{Code: "IAM.User.NotFound"}) {
		t.Fatal("expected .NotFound suffix to match")
	}
	if IsNotFound(&APIError{Code: "IAM.Invalid"}) {
		t.Fatal("unexpected IsNotFound match")
	}
}

func TestIsAccessDenied(t *testing.T) {
	if !IsAccessDenied(&APIError{Code: "IAM.0002", Message: "forbidden", StatusCode: 403}) {
		t.Fatal("expected access denied match")
	}
}
