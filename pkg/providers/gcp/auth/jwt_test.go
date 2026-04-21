package auth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func TestSignAssertionPKCS8(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cred := Credential{
		ProjectID:     "demo-project",
		PrivateKeyID:  "kid-1",
		PrivateKeyPEM: testutil.PKCS8PrivateKeyPEM,
		ClientEmail:   "demo@example.com",
		TokenURI:      "https://oauth2.googleapis.com/token",
		Scopes:        []string{"scope-a", "scope-b"},
	}

	assertion, err := SignAssertion(cred, now)
	if err != nil {
		t.Fatalf("SignAssertion() error = %v", err)
	}

	parts := strings.Split(assertion, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected jwt parts: %d", len(parts))
	}

	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if string(header) != `{"alg":"RS256","typ":"JWT","kid":"kid-1"}` {
		t.Fatalf("unexpected header json: %s", string(header))
	}

	claims, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	if string(claims) != `{"iss":"demo@example.com","scope":"scope-a scope-b","aud":"https://oauth2.googleapis.com/token","iat":1700000000,"exp":1700003600}` {
		t.Fatalf("unexpected claims json: %s", string(claims))
	}

	publicKey := parseRSAPublicKey(t, testutil.RSAPublicKeyPEM)
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, sum[:], signature); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
}

func TestSignAssertionPKCS1(t *testing.T) {
	cred := Credential{
		ProjectID:     "demo-project",
		PrivateKeyPEM: testutil.PKCS1PrivateKeyPEM,
		ClientEmail:   "demo@example.com",
		TokenURI:      "https://oauth2.googleapis.com/token",
		Scopes:        []string{DefaultScope},
	}
	if _, err := SignAssertion(cred, time.Unix(1700000000, 0)); err != nil {
		t.Fatalf("SignAssertion() error = %v", err)
	}
}

func TestParseRSAPrivateKeyRejectsEC(t *testing.T) {
	if _, err := ParseRSAPrivateKey(testutil.ECPrivateKeyPEM); err == nil || !strings.Contains(err.Error(), `unsupported private key type "EC PRIVATE KEY"`) {
		t.Fatalf("expected unsupported key type error, got %v", err)
	}
}

func parseRSAPublicKey(t *testing.T, pemData string) *rsa.PublicKey {
	t.Helper()
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		t.Fatal("public key pem decode failed")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("parse public key: %v", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("unexpected public key type: %T", key)
	}
	return rsaKey
}
