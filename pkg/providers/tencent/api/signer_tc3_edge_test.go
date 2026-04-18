package api

import (
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

// These are maintainer-added regression guards for cases the captured
// fixtures don't exercise. Each one encodes an invariant a future refactor
// might accidentally break.
//
// Reference: TC3 algorithm at
// https://cloud.tencent.com/document/api/213/30654

func TestSigner_EmptyServiceRejected(t *testing.T) {
	_, err := TC3Signer{}.Sign(auth.New("AK", "SK", ""), SignInput{
		Host: "x.tencentcloudapi.com", Method: "POST",
	})
	if err == nil {
		t.Fatal("expected error for empty service")
	}
}

func TestSigner_EmptyHostRejected(t *testing.T) {
	_, err := TC3Signer{}.Sign(auth.New("AK", "SK", ""), SignInput{
		Service: "cvm", Method: "POST",
	})
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestSigner_InvalidCredentialRejected(t *testing.T) {
	_, err := TC3Signer{}.Sign(auth.New("", "SK", ""), SignInput{
		Service: "cvm", Host: "cvm.tencentcloudapi.com", Method: "POST",
	})
	if err == nil {
		t.Fatal("expected error for empty secret id")
	}
	_, err = TC3Signer{}.Sign(auth.New("AK", "", ""), SignInput{
		Service: "cvm", Host: "cvm.tencentcloudapi.com", Method: "POST",
	})
	if err == nil {
		t.Fatal("expected error for empty secret key")
	}
}

// Empty payload must hash as SHA256("") per the SDK. Regressions where an
// empty body is pre-normalised to "{}" would break every service that sends
// GET-style (no body) requests in the future.
func TestSigner_EmptyPayloadHashesAsEmpty(t *testing.T) {
	const sha256OfEmpty = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	got, err := TC3Signer{}.Sign(auth.New("AK", "SK", ""), SignInput{
		Service: "cvm", Host: "cvm.tencentcloudapi.com",
		Method: "POST", ContentType: "application/json",
		Timestamp: time.Unix(1700000000, 0).UTC(),
		// Payload intentionally nil — the signer must NOT synthesise "{}".
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if got.PayloadHash != sha256OfEmpty {
		t.Fatalf("empty payload hash = %s, want %s", got.PayloadHash, sha256OfEmpty)
	}
}

// TC3 unsigned-payload mode replaces the payload hash with the literal
// SHA256 of the string "UNSIGNED-PAYLOAD". A refactor that forgets this
// branch would produce silently incorrect signatures for any future service
// that opts into unsigned payloads (streaming, large uploads).
func TestSigner_UnsignedPayloadSentinel(t *testing.T) {
	const unsignedHash = "438d4109ef0d676b8c2c7ed13cdfcb418e494d53b843d4634ce3b1085f07bb96"
	got, err := TC3Signer{}.Sign(auth.New("AK", "SK", ""), SignInput{
		Service: "cvm", Host: "cvm.tencentcloudapi.com",
		Method: "POST", ContentType: "application/json",
		Timestamp:       time.Unix(1700000000, 0).UTC(),
		Payload:         []byte(`{"this":"is ignored"}`),
		UnsignedPayload: true,
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if got.PayloadHash != unsignedHash {
		t.Fatalf("unsigned payload hash = %s, want %s", got.PayloadHash, unsignedHash)
	}
}

// Timestamps landing on a day boundary must yield the correct date component
// in the credential scope. Off-by-one in UTC conversion would only surface
// on requests made within ±8h of UTC midnight, so a captured fixture
// generated at an arbitrary time is unlikely to catch it.
func TestSigner_DateBoundary(t *testing.T) {
	// 2024-01-01 00:00:00 UTC exactly.
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := TC3Signer{}.Sign(auth.New("AK", "SK", ""), SignInput{
		Service: "cvm", Host: "cvm.tencentcloudapi.com",
		Method: "POST", ContentType: "application/json",
		Timestamp: ts, Payload: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if want := "2024-01-01/cvm/tc3_request"; got.CredentialScope != want {
		t.Fatalf("credential scope = %s, want %s", got.CredentialScope, want)
	}

	// 2023-12-31 23:59:59 UTC — same day in UTC even if the caller's local
	// clock already flipped to 2024.
	ts = time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)
	got, err = TC3Signer{}.Sign(auth.New("AK", "SK", ""), SignInput{
		Service: "cvm", Host: "cvm.tencentcloudapi.com",
		Method: "POST", ContentType: "application/json",
		Timestamp: ts, Payload: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if want := "2023-12-31/cvm/tc3_request"; got.CredentialScope != want {
		t.Fatalf("credential scope = %s, want %s", got.CredentialScope, want)
	}
}

// Non-UTC inputs must be converted to UTC before date extraction. A naive
// .Format("2006-01-02") on a local-time input would leak the caller's
// timezone into the credential scope.
func TestSigner_NonUTCTimestampNormalised(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	ts := time.Date(2024, 1, 1, 7, 0, 0, 0, loc) // 2023-12-31 23:00 UTC
	got, err := TC3Signer{}.Sign(auth.New("AK", "SK", ""), SignInput{
		Service: "cvm", Host: "cvm.tencentcloudapi.com",
		Method: "POST", ContentType: "application/json",
		Timestamp: ts, Payload: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if want := "2023-12-31/cvm/tc3_request"; got.CredentialScope != want {
		t.Fatalf("credential scope = %s, want %s (UTC conversion missing)", got.CredentialScope, want)
	}
}

// The Authorization header is the sole contract with the API; callers will
// produce it verbatim. This asserts the literal format so a stray space /
// missing comma / rearranged field order is caught immediately.
func TestSigner_AuthorizationFormat(t *testing.T) {
	got, err := TC3Signer{}.Sign(auth.New("AKIDfmt", "SK", ""), SignInput{
		Service: "cvm", Host: "cvm.tencentcloudapi.com",
		Method: "POST", ContentType: "application/json",
		Timestamp: time.Unix(1700000000, 0).UTC(),
		Payload:   []byte("{}"),
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !strings.HasPrefix(got.Authorization, "TC3-HMAC-SHA256 Credential=AKIDfmt/") {
		t.Errorf("prefix mismatch: %s", got.Authorization)
	}
	if !strings.Contains(got.Authorization, ", SignedHeaders=content-type;host, Signature=") {
		t.Errorf("middle mismatch: %s", got.Authorization)
	}
	// Signature is 64 hex chars.
	idx := strings.LastIndex(got.Authorization, "Signature=")
	if idx < 0 {
		t.Fatalf("no signature: %s", got.Authorization)
	}
	if sig := got.Authorization[idx+len("Signature="):]; len(sig) != 64 {
		t.Errorf("signature length = %d, want 64 hex chars", len(sig))
	}
}
