package api

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignerMatchesFixtures(t *testing.T) {
	for _, fx := range signerFixtures() {
		fx := fx
		t.Run(fx.Name, func(t *testing.T) {
			got, err := Sign(&SignRequest{
				Method:    fx.Input.Method,
				Host:      fx.Input.Host,
				Path:      fx.Input.Path,
				Query:     mustParseSignerQuery(t, fx.Input.Query),
				Headers:   fixtureHeaders(fx.Input),
				Body:      []byte(fx.Input.Body),
				AccessKey: fx.Input.AccessKey,
				SecretKey: fx.Input.SecretKey,
				Timestamp: mustParseXSdkDate(t, fx.Input.XSdkDate),
			})
			if err != nil {
				t.Fatalf("Sign() error = %v", err)
			}
			if got[HeaderAuthorization] != fx.Expected.Authorization {
				signedHeaders, canonicalRequest, stringToSign := deriveSignerDiagnostics(t, fx.Input)
				t.Fatalf(
					"authorization mismatch\n got:  %s\nwant: %s\nsigned_headers got=%s want=%s\ncanonical_match=%v\nstring_to_sign_match=%v",
					got[HeaderAuthorization],
					fx.Expected.Authorization,
					signedHeaders,
					fx.Expected.SignedHeaders,
					canonicalRequest == fx.Expected.CanonicalRequest,
					stringToSign == fx.Expected.StringToSign,
				)
			}
			if got[HeaderXDate] != fx.Input.XSdkDate {
				t.Fatalf("unexpected x-sdk-date: %s", got[HeaderXDate])
			}
			if got[HeaderContentSha256] != "" {
				t.Fatalf("unexpected %s header: %s", HeaderContentSha256, got[HeaderContentSha256])
			}
			signedHeaders, canonicalRequest, stringToSign := deriveSignerDiagnostics(t, fx.Input)
			if signedHeaders != fx.Expected.SignedHeaders {
				t.Fatalf("signed headers mismatch\n got:  %s\nwant: %s", signedHeaders, fx.Expected.SignedHeaders)
			}
			if canonicalRequest != fx.Expected.CanonicalRequest {
				t.Fatalf("canonical request mismatch\n got:  %s\nwant: %s", canonicalRequest, fx.Expected.CanonicalRequest)
			}
			if stringToSign != fx.Expected.StringToSign {
				t.Fatalf("string to sign mismatch\n got:  %s\nwant: %s", stringToSign, fx.Expected.StringToSign)
			}
		})
	}
}

func deriveSignerDiagnostics(t *testing.T, input signerFixtureInput) (string, string, string) {
	t.Helper()
	headers := fixtureHeaders(input)
	headers[strings.ToLower(HeaderXDate)] = input.XSdkDate
	signedHeaders := strings.Join(signedHeaderNames(normalizeHeaders(headers)), ";")
	canonicalRequest := strings.Join([]string{
		strings.ToUpper(input.Method),
		canonicalURI(input.Path),
		canonicalQueryString(mustParseSignerQuery(t, input.Query)),
		canonicalHeaders(normalizeHeaders(headers), signedHeaderNames(normalizeHeaders(headers))),
		signedHeaders,
		hexEncodeSHA256Hash([]byte(input.Body)),
	}, "\n")
	stringToSign := strings.Join([]string{
		Algorithm,
		input.XSdkDate,
		hexEncodeSHA256Hash([]byte(canonicalRequest)),
	}, "\n")
	return signedHeaders, canonicalRequest, stringToSign
}

func fixtureHeaders(input signerFixtureInput) map[string]string {
	headers := make(map[string]string, len(input.ExtraHeaders)+1)
	if input.ContentType != "" {
		headers["Content-Type"] = input.ContentType
	}
	for key, value := range input.ExtraHeaders {
		headers[key] = value
	}
	return headers
}

func mustParseXSdkDate(t *testing.T, raw string) time.Time {
	t.Helper()
	ts, err := time.Parse(BasicDateFormat, raw)
	if err != nil {
		t.Fatalf("parse x-sdk-date: %v", err)
	}
	return ts
}

func mustParseSignerQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	if raw == "" {
		return nil
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	return values
}
