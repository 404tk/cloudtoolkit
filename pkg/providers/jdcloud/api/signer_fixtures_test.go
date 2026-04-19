package api

import (
	"net/http"
	"testing"
	"time"
)

type signerFixture struct {
	name     string
	input    SignInput
	expected Signature
}

func TestSignerMatchesFixtures(t *testing.T) {
	for _, fx := range signerFixtures() {
		fx := fx
		t.Run(fx.name, func(t *testing.T) {
			got, err := Sign(fx.input)
			if err != nil {
				t.Fatalf("Sign() error = %v", err)
			}
			assertSignatureField(t, "Authorization", got.Authorization, fx.expected.Authorization)
			assertSignatureField(t, "SignedHeaders", got.SignedHeaders, fx.expected.SignedHeaders)
			assertSignatureField(t, "CredentialScope", got.CredentialScope, fx.expected.CredentialScope)
			assertSignatureField(t, "CanonicalRequest", got.CanonicalRequest, fx.expected.CanonicalRequest)
			assertSignatureField(t, "StringToSign", got.StringToSign, fx.expected.StringToSign)
			assertSignatureField(t, "XJdcloudDate", got.XJdcloudDate, fx.expected.XJdcloudDate)
			assertSignatureField(t, "XJdcloudNonce", got.XJdcloudNonce, fx.expected.XJdcloudNonce)
			assertSignatureField(t, "BodyDigest", got.BodyDigest, fx.expected.BodyDigest)
		})
	}
}

func signerFixtures() []signerFixture {
	ts := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	return []signerFixture{
		{
			name: "iam_describe_sub_users",
			input: SignInput{
				Method:       http.MethodGet,
				Host:         "iam.jdcloud-api.com",
				Path:         "/v1/subUsers",
				ContentType:  "application/json",
				Service:      "iam",
				Region:       "jdcloud-api",
				AccessKey:    "JDCsigcaptureAKID",
				SecretKey:    "JDCsigcaptureSECRET1234567890abcdefg",
				SessionToken: "SkRDc2lnY2FwdHVyZVRPS0VOMDAxMjM0NTY3ODkw",
				Nonce:        "ebf8b26d-c3be-402f-9f10-f8b6573fd823",
				Timestamp:    ts,
			},
			expected: Signature{
				Authorization:    "JDCLOUD2-HMAC-SHA256 Credential=JDCsigcaptureAKID/20260419/jdcloud-api/iam/jdcloud2_request, SignedHeaders=content-type;host;x-jdcloud-date;x-jdcloud-nonce;x-jdcloud-security-token, Signature=64ad103670ecf8761b9b6c1313de8a987d5ea292e78b0ef8833f84ba30936c1e",
				SignedHeaders:    "content-type;host;x-jdcloud-date;x-jdcloud-nonce;x-jdcloud-security-token",
				CredentialScope:  "20260419/jdcloud-api/iam/jdcloud2_request",
				CanonicalRequest: "GET\n/v1/subUsers\n\ncontent-type:application/json\nhost:iam.jdcloud-api.com\nx-jdcloud-date:20260419T120000Z\nx-jdcloud-nonce:ebf8b26d-c3be-402f-9f10-f8b6573fd823\nx-jdcloud-security-token:SkRDc2lnY2FwdHVyZVRPS0VOMDAxMjM0NTY3ODkw\n\ncontent-type;host;x-jdcloud-date;x-jdcloud-nonce;x-jdcloud-security-token\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "JDCLOUD2-HMAC-SHA256\n20260419T120000Z\n20260419/jdcloud-api/iam/jdcloud2_request\ne4e281a48a6356cafb2c387563cfa1dce328907c03ac44cbf4cfe6b802f19bfe",
				XJdcloudDate:     "20260419T120000Z",
				XJdcloudNonce:    "ebf8b26d-c3be-402f-9f10-f8b6573fd823",
				BodyDigest:       emptyBodySHA256Hex,
			},
		},
		{
			name: "iam_describe_sub_user",
			input: SignInput{
				Method:      http.MethodGet,
				Host:        "iam.jdcloud-api.com",
				Path:        "/v1/subUsers/ctk-subuser",
				ContentType: "application/json",
				Service:     "iam",
				Region:      "jdcloud-api",
				AccessKey:   "JDCsigcaptureAKID",
				SecretKey:   "JDCsigcaptureSECRET1234567890abcdefg",
				Nonce:       "cd762f60-84d8-45f2-8455-f3e0bf348305",
				Timestamp:   ts,
			},
			expected: Signature{
				Authorization:    "JDCLOUD2-HMAC-SHA256 Credential=JDCsigcaptureAKID/20260419/jdcloud-api/iam/jdcloud2_request, SignedHeaders=content-type;host;x-jdcloud-date;x-jdcloud-nonce, Signature=7f5dafd3fcc4fec3fdd395830c535c9ebfdefc528d39b547c74525d1d55447b9",
				SignedHeaders:    "content-type;host;x-jdcloud-date;x-jdcloud-nonce",
				CredentialScope:  "20260419/jdcloud-api/iam/jdcloud2_request",
				CanonicalRequest: "GET\n/v1/subUsers/ctk-subuser\n\ncontent-type:application/json\nhost:iam.jdcloud-api.com\nx-jdcloud-date:20260419T120000Z\nx-jdcloud-nonce:cd762f60-84d8-45f2-8455-f3e0bf348305\n\ncontent-type;host;x-jdcloud-date;x-jdcloud-nonce\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "JDCLOUD2-HMAC-SHA256\n20260419T120000Z\n20260419/jdcloud-api/iam/jdcloud2_request\n91ad4c89252154f0fed407d8550524b5e71cd8f6c47f1b54cc24c5d29679345b",
				XJdcloudDate:     "20260419T120000Z",
				XJdcloudNonce:    "cd762f60-84d8-45f2-8455-f3e0bf348305",
				BodyDigest:       emptyBodySHA256Hex,
			},
		},
		{
			name: "vm_describe_instances",
			input: SignInput{
				Method:       http.MethodGet,
				Host:         "vm.jdcloud-api.com",
				Path:         "/v1/regions/cn-north-1/instances",
				ContentType:  "application/json",
				Service:      "vm",
				Region:       "cn-north-1",
				AccessKey:    "JDCsigcaptureAKID",
				SecretKey:    "JDCsigcaptureSECRET1234567890abcdefg",
				SessionToken: "SkRDc2lnY2FwdHVyZVRPS0VOMDAxMjM0NTY3ODkw",
				Nonce:        "6691f002-da59-46eb-882c-edb66d46c917",
				Timestamp:    ts,
			},
			expected: Signature{
				Authorization:    "JDCLOUD2-HMAC-SHA256 Credential=JDCsigcaptureAKID/20260419/cn-north-1/vm/jdcloud2_request, SignedHeaders=content-type;host;x-jdcloud-date;x-jdcloud-nonce;x-jdcloud-security-token, Signature=603defafc3dc3ace2e602d3629279e8caffab622948e1b3ca6407bbb9c8e8c5e",
				SignedHeaders:    "content-type;host;x-jdcloud-date;x-jdcloud-nonce;x-jdcloud-security-token",
				CredentialScope:  "20260419/cn-north-1/vm/jdcloud2_request",
				CanonicalRequest: "GET\n/v1/regions/cn-north-1/instances\n\ncontent-type:application/json\nhost:vm.jdcloud-api.com\nx-jdcloud-date:20260419T120000Z\nx-jdcloud-nonce:6691f002-da59-46eb-882c-edb66d46c917\nx-jdcloud-security-token:SkRDc2lnY2FwdHVyZVRPS0VOMDAxMjM0NTY3ODkw\n\ncontent-type;host;x-jdcloud-date;x-jdcloud-nonce;x-jdcloud-security-token\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "JDCLOUD2-HMAC-SHA256\n20260419T120000Z\n20260419/cn-north-1/vm/jdcloud2_request\nf9d64d328a07a7b712bd3bbc6ffb949a5df61068a41e1dd984cd2ad663d6e028",
				XJdcloudDate:     "20260419T120000Z",
				XJdcloudNonce:    "6691f002-da59-46eb-882c-edb66d46c917",
				BodyDigest:       emptyBodySHA256Hex,
			},
		},
		{
			name: "oss_list_buckets",
			input: SignInput{
				Method:      http.MethodGet,
				Host:        "oss.jdcloud-api.com",
				Path:        "/v1/regions/cn-north-1/buckets",
				ContentType: "application/json",
				Service:     "oss",
				Region:      "cn-north-1",
				AccessKey:   "JDCsigcaptureAKID",
				SecretKey:   "JDCsigcaptureSECRET1234567890abcdefg",
				Nonce:       "3233e986-8ad0-41b6-a9f7-83b052dc5577",
				Timestamp:   ts,
			},
			expected: Signature{
				Authorization:    "JDCLOUD2-HMAC-SHA256 Credential=JDCsigcaptureAKID/20260419/cn-north-1/oss/jdcloud2_request, SignedHeaders=content-type;host;x-jdcloud-date;x-jdcloud-nonce, Signature=51b58f6f79ea48e3ec681fff38842884d3346e4ecfed83db32307e1f26766865",
				SignedHeaders:    "content-type;host;x-jdcloud-date;x-jdcloud-nonce",
				CredentialScope:  "20260419/cn-north-1/oss/jdcloud2_request",
				CanonicalRequest: "GET\n/v1/regions/cn-north-1/buckets\n\ncontent-type:application/json\nhost:oss.jdcloud-api.com\nx-jdcloud-date:20260419T120000Z\nx-jdcloud-nonce:3233e986-8ad0-41b6-a9f7-83b052dc5577\n\ncontent-type;host;x-jdcloud-date;x-jdcloud-nonce\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "JDCLOUD2-HMAC-SHA256\n20260419T120000Z\n20260419/cn-north-1/oss/jdcloud2_request\nc38f31f80b249ceb5d25f7de2e6d7b138aa9b0010d0a8dc4fd43e5d8b59fc06b",
				XJdcloudDate:     "20260419T120000Z",
				XJdcloudNonce:    "3233e986-8ad0-41b6-a9f7-83b052dc5577",
				BodyDigest:       emptyBodySHA256Hex,
			},
		},
	}
}

func assertSignatureField(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s mismatch\n got: %s\nwant: %s", field, got, want)
	}
}
