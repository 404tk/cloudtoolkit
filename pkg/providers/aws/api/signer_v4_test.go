package api

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

type signerGoldenFixture struct {
	name       string
	credential auth.Credential
	input      SignInput
	expected   Signature
}

func TestSigV4SignerGoldenFixtures(t *testing.T) {
	fixtures := []signerGoldenFixture{
		{
			name:       "sts_getcalleridentity_global",
			credential: auth.New("AKIDAWSsigcaptureglobal", "SECRETawsSigCaptureGlobal1234567890", ""),
			input: SignInput{
				Method:      "POST",
				Service:     "sts",
				Region:      "us-east-1",
				Host:        "sts.us-east-1.amazonaws.com",
				Path:        "/",
				Query:       url.Values{},
				ContentType: "application/x-www-form-urlencoded",
				Payload:     []byte("Action=GetCallerIdentity&Version=2011-06-15"),
				Timestamp:   mustParseFixtureTime(t, "20260418T145530Z"),
				Headers: http.Header{
					"Amz-Sdk-Invocation-Id": []string{"d92927e8-8d31-42b1-addd-ef1d913933d4"},
					"Amz-Sdk-Request":       []string{"attempt=1; max=3"},
				},
			},
			expected: Signature{
				Authorization:    "AWS4-HMAC-SHA256 Credential=AKIDAWSsigcaptureglobal/20260418/us-east-1/sts/aws4_request, SignedHeaders=amz-sdk-invocation-id;amz-sdk-request;content-length;content-type;host;x-amz-date, Signature=99e24335a37888dfd70ccf4c279df7a596891f46aec3f04ccb6ac7e5c6f152d4",
				SignedHeaders:    "amz-sdk-invocation-id;amz-sdk-request;content-length;content-type;host;x-amz-date",
				CredentialScope:  "20260418/us-east-1/sts/aws4_request",
				CanonicalRequest: "POST\n/\n\namz-sdk-invocation-id:d92927e8-8d31-42b1-addd-ef1d913933d4\namz-sdk-request:attempt=1; max=3\ncontent-length:43\ncontent-type:application/x-www-form-urlencoded\nhost:sts.us-east-1.amazonaws.com\nx-amz-date:20260418T145530Z\n\namz-sdk-invocation-id;amz-sdk-request;content-length;content-type;host;x-amz-date\nab821ae955788b0e33ebd34c208442ccfc2d406e2edc5e7a39bd6458fbb4f843",
				StringToSign:     "AWS4-HMAC-SHA256\n20260418T145530Z\n20260418/us-east-1/sts/aws4_request\neb91b4e7899efba2e7bb7aa1fbdc533739153646947e3679387024d735401b78",
				AmzDate:          "20260418T145530Z",
				PayloadHash:      "ab821ae955788b0e33ebd34c208442ccfc2d406e2edc5e7a39bd6458fbb4f843",
			},
		},
		{
			name:       "sts_getcalleridentity_cn",
			credential: auth.New("AKIDAWSsigcapturechina", "SECRETawsSigCaptureChina1234567890", "TOKENawsSigCaptureChina1234567890"),
			input: SignInput{
				Method:      "POST",
				Service:     "sts",
				Region:      "cn-northwest-1",
				Host:        "sts.cn-northwest-1.amazonaws.com.cn",
				Path:        "/",
				Query:       url.Values{},
				ContentType: "application/x-www-form-urlencoded",
				Payload:     []byte("Action=GetCallerIdentity&Version=2011-06-15"),
				Timestamp:   mustParseFixtureTime(t, "20260418T145530Z"),
				Headers: http.Header{
					"Amz-Sdk-Invocation-Id": []string{"1e3348e7-922d-437c-8201-a1c33d382b31"},
					"Amz-Sdk-Request":       []string{"attempt=1; max=3"},
				},
			},
			expected: Signature{
				Authorization:    "AWS4-HMAC-SHA256 Credential=AKIDAWSsigcapturechina/20260418/cn-northwest-1/sts/aws4_request, SignedHeaders=amz-sdk-invocation-id;amz-sdk-request;content-length;content-type;host;x-amz-date;x-amz-security-token, Signature=2ac710de215dd8baba895d0b341e74622cfdc5ce77d4ad92f54b5cd0e2667852",
				SignedHeaders:    "amz-sdk-invocation-id;amz-sdk-request;content-length;content-type;host;x-amz-date;x-amz-security-token",
				CredentialScope:  "20260418/cn-northwest-1/sts/aws4_request",
				CanonicalRequest: "POST\n/\n\namz-sdk-invocation-id:1e3348e7-922d-437c-8201-a1c33d382b31\namz-sdk-request:attempt=1; max=3\ncontent-length:43\ncontent-type:application/x-www-form-urlencoded\nhost:sts.cn-northwest-1.amazonaws.com.cn\nx-amz-date:20260418T145530Z\nx-amz-security-token:TOKENawsSigCaptureChina1234567890\n\namz-sdk-invocation-id;amz-sdk-request;content-length;content-type;host;x-amz-date;x-amz-security-token\nab821ae955788b0e33ebd34c208442ccfc2d406e2edc5e7a39bd6458fbb4f843",
				StringToSign:     "AWS4-HMAC-SHA256\n20260418T145530Z\n20260418/cn-northwest-1/sts/aws4_request\n391df7d8bb65e311fe5c4bc1069a3ffaa7e6d006e67024d4f99dd8676bffaa22",
				AmzDate:          "20260418T145530Z",
				PayloadHash:      "ab821ae955788b0e33ebd34c208442ccfc2d406e2edc5e7a39bd6458fbb4f843",
			},
		},
	}

	signer := SigV4Signer{}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			got, err := signer.Sign(fixture.credential, fixture.input)
			if err != nil {
				t.Fatalf("Sign() error = %v", err)
			}
			assertSignatureField(t, "Authorization", got.Authorization, fixture.expected.Authorization)
			assertSignatureField(t, "SignedHeaders", got.SignedHeaders, fixture.expected.SignedHeaders)
			assertSignatureField(t, "CredentialScope", got.CredentialScope, fixture.expected.CredentialScope)
			assertSignatureField(t, "CanonicalRequest", got.CanonicalRequest, fixture.expected.CanonicalRequest)
			assertSignatureField(t, "StringToSign", got.StringToSign, fixture.expected.StringToSign)
			assertSignatureField(t, "AmzDate", got.AmzDate, fixture.expected.AmzDate)
			assertSignatureField(t, "PayloadHash", got.PayloadHash, fixture.expected.PayloadHash)
		})
	}
}

func assertSignatureField(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s mismatch\n got: %s\nwant: %s", field, got, want)
	}
}

func mustParseFixtureTime(t *testing.T, value string) time.Time {
	t.Helper()
	ts, err := time.Parse("20060102T150405Z", value)
	if err != nil {
		t.Fatalf("parse fixture time %q: %v", value, err)
	}
	return ts
}
