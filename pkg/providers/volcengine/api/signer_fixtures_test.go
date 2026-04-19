package api

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

type signerFixture struct {
	name     string
	input    SignInput
	expected Signature
}

func TestSignerMatchesFixtures(t *testing.T) {
	for _, fx := range signerFixtures(t) {
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
			assertSignatureField(t, "XDate", got.XDate, fx.expected.XDate)
			assertSignatureField(t, "XContentSHA256", got.XContentSHA256, fx.expected.XContentSHA256)
		})
	}
}

func signerFixtures(t *testing.T) []signerFixture {
	ts := mustParseFixtureTime(t, "20260419T120000Z")
	return []signerFixture{
		{
			name: "iam_list_projects",
			input: SignInput{
				Method:      http.MethodGet,
				Host:        "iam.volcengineapi.com",
				Path:        "/",
				Query:       mustParseSignerQuery(t, "Action=ListProjects&Version=2021-08-01"),
				Body:        nil,
				ContentType: "application/x-www-form-urlencoded; charset=utf-8",
				Service:     "iam",
				Region:      "cn-beijing",
				AccessKey:   "VOLCsigcaptureAKID",
				SecretKey:   "VOLCsigcaptureSECRET1234567890abcdefg",
				Timestamp:   ts,
			},
			expected: Signature{
				Authorization:    "HMAC-SHA256 Credential=VOLCsigcaptureAKID/20260419/cn-beijing/iam/request, SignedHeaders=content-type;host;x-content-sha256;x-date, Signature=d883ff67578d5f0e1c8be8a8641dfc764f636615821f4673e205babfbeb6d051",
				SignedHeaders:    "content-type;host;x-content-sha256;x-date",
				CredentialScope:  "20260419/cn-beijing/iam/request",
				CanonicalRequest: "GET\n/\nAction=ListProjects&Version=2021-08-01\ncontent-type:application/x-www-form-urlencoded; charset=utf-8\nhost:iam.volcengineapi.com\nx-content-sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nx-date:20260419T120000Z\n\ncontent-type;host;x-content-sha256;x-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "HMAC-SHA256\n20260419T120000Z\n20260419/cn-beijing/iam/request\n25c9e2996fb082d4dc5d8886c70a94cf6d7e50b8f45b766964ba82b4080a8c9e",
				XDate:            "20260419T120000Z",
				XContentSHA256:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		},
		{
			name: "iam_list_users",
			input: SignInput{
				Method:      http.MethodGet,
				Host:        "iam.volcengineapi.com",
				Path:        "/",
				Query:       mustParseSignerQuery(t, "Action=ListUsers&Limit=100&Offset=0&Version=2018-01-01"),
				Body:        nil,
				ContentType: "application/x-www-form-urlencoded; charset=utf-8",
				Service:     "iam",
				Region:      "cn-beijing",
				AccessKey:   "VOLCsigcaptureAKID",
				SecretKey:   "VOLCsigcaptureSECRET1234567890abcdefg",
				Timestamp:   ts,
			},
			expected: Signature{
				Authorization:    "HMAC-SHA256 Credential=VOLCsigcaptureAKID/20260419/cn-beijing/iam/request, SignedHeaders=content-type;host;x-content-sha256;x-date, Signature=76504c87d4a03f82068be6fa4bc02c64498b1b2759a0ed9a310692035c51ab7d",
				SignedHeaders:    "content-type;host;x-content-sha256;x-date",
				CredentialScope:  "20260419/cn-beijing/iam/request",
				CanonicalRequest: "GET\n/\nAction=ListUsers&Limit=100&Offset=0&Version=2018-01-01\ncontent-type:application/x-www-form-urlencoded; charset=utf-8\nhost:iam.volcengineapi.com\nx-content-sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nx-date:20260419T120000Z\n\ncontent-type;host;x-content-sha256;x-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "HMAC-SHA256\n20260419T120000Z\n20260419/cn-beijing/iam/request\n552d2bc60332b72bf237ef00fdbd09444a74a9b63cd84d353e64a05cd78dc89e",
				XDate:            "20260419T120000Z",
				XContentSHA256:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		},
		{
			name: "iam_get_login_profile",
			input: SignInput{
				Method:       http.MethodGet,
				Host:         "iam.volcengineapi.com",
				Path:         "/",
				Query:        mustParseSignerQuery(t, "Action=GetLoginProfile&UserName=ctk&Version=2018-01-01"),
				Body:         nil,
				ContentType:  "application/x-www-form-urlencoded; charset=utf-8",
				Service:      "iam",
				Region:       "cn-beijing",
				AccessKey:    "VOLCsigcaptureAKID",
				SecretKey:    "VOLCsigcaptureSECRET1234567890abcdefg",
				SessionToken: "VOLCsigcaptureTOKEN789",
				Timestamp:    ts,
			},
			expected: Signature{
				Authorization:    "HMAC-SHA256 Credential=VOLCsigcaptureAKID/20260419/cn-beijing/iam/request, SignedHeaders=content-type;host;x-content-sha256;x-date;x-security-token, Signature=5e54b48114171f4bafedfab3af271813ef028bb9ad2cc63cb0c3266ffcbb0439",
				SignedHeaders:    "content-type;host;x-content-sha256;x-date;x-security-token",
				CredentialScope:  "20260419/cn-beijing/iam/request",
				CanonicalRequest: "GET\n/\nAction=GetLoginProfile&UserName=ctk&Version=2018-01-01\ncontent-type:application/x-www-form-urlencoded; charset=utf-8\nhost:iam.volcengineapi.com\nx-content-sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nx-date:20260419T120000Z\nx-security-token:VOLCsigcaptureTOKEN789\n\ncontent-type;host;x-content-sha256;x-date;x-security-token\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "HMAC-SHA256\n20260419T120000Z\n20260419/cn-beijing/iam/request\n268aa042c07ec62cad626db0d0d0a41ed26f03a558f4b63f89348257eb7b0647",
				XDate:            "20260419T120000Z",
				XContentSHA256:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		},
		{
			name: "billing_query_balance_acct",
			input: SignInput{
				Method:      http.MethodPost,
				Host:        "billing.volcengineapi.com",
				Path:        "/",
				Query:       mustParseSignerQuery(t, "Action=QueryBalanceAcct&Version=2022-01-01"),
				Body:        []byte("{}"),
				ContentType: "application/json; charset=utf-8",
				Service:     "billing",
				Region:      "cn-beijing",
				AccessKey:   "VOLCsigcaptureAKID",
				SecretKey:   "VOLCsigcaptureSECRET1234567890abcdefg",
				Timestamp:   ts,
			},
			expected: Signature{
				Authorization:    "HMAC-SHA256 Credential=VOLCsigcaptureAKID/20260419/cn-beijing/billing/request, SignedHeaders=content-type;host;x-content-sha256;x-date, Signature=17e9a2a1fb6212e819792c787167a93dac27212a018d3fbf89cf128b66785983",
				SignedHeaders:    "content-type;host;x-content-sha256;x-date",
				CredentialScope:  "20260419/cn-beijing/billing/request",
				CanonicalRequest: "POST\n/\nAction=QueryBalanceAcct&Version=2022-01-01\ncontent-type:application/json; charset=utf-8\nhost:billing.volcengineapi.com\nx-content-sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a\nx-date:20260419T120000Z\n\ncontent-type;host;x-content-sha256;x-date\n44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
				StringToSign:     "HMAC-SHA256\n20260419T120000Z\n20260419/cn-beijing/billing/request\na85f0d1bb730a0aee4c089cae63b1104417c4ac6fe2310d9e1afa1985ad15639",
				XDate:            "20260419T120000Z",
				XContentSHA256:   "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
			},
		},
		{
			name: "ecs_describe_regions",
			input: SignInput{
				Method:      http.MethodGet,
				Host:        "ecs.cn-beijing.volcengineapi.com",
				Path:        "/",
				Query:       mustParseSignerQuery(t, "Action=DescribeRegions&MaxResults=100&Version=2020-04-01"),
				Body:        nil,
				ContentType: "application/x-www-form-urlencoded; charset=utf-8",
				Service:     "ecs",
				Region:      "cn-beijing",
				AccessKey:   "VOLCsigcaptureAKID",
				SecretKey:   "VOLCsigcaptureSECRET1234567890abcdefg",
				Timestamp:   ts,
			},
			expected: Signature{
				Authorization:    "HMAC-SHA256 Credential=VOLCsigcaptureAKID/20260419/cn-beijing/ecs/request, SignedHeaders=content-type;host;x-content-sha256;x-date, Signature=1c625e7c3f6a93c6921d4defd6aef2e66751a23484c5b8bd44f601d03cdda561",
				SignedHeaders:    "content-type;host;x-content-sha256;x-date",
				CredentialScope:  "20260419/cn-beijing/ecs/request",
				CanonicalRequest: "GET\n/\nAction=DescribeRegions&MaxResults=100&Version=2020-04-01\ncontent-type:application/x-www-form-urlencoded; charset=utf-8\nhost:ecs.cn-beijing.volcengineapi.com\nx-content-sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nx-date:20260419T120000Z\n\ncontent-type;host;x-content-sha256;x-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "HMAC-SHA256\n20260419T120000Z\n20260419/cn-beijing/ecs/request\n3acf2c3681ef20f4e8ba45fc4251e03ef3067c7af5928cc8f4ddfcdf9fa54a5d",
				XDate:            "20260419T120000Z",
				XContentSHA256:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		},
		{
			name: "ecs_describe_instances",
			input: SignInput{
				Method:      http.MethodGet,
				Host:        "ecs.cn-beijing.volcengineapi.com",
				Path:        "/",
				Query:       mustParseSignerQuery(t, "Action=DescribeInstances&MaxResults=100&NextToken=&Version=2020-04-01"),
				Body:        nil,
				ContentType: "application/x-www-form-urlencoded; charset=utf-8",
				Service:     "ecs",
				Region:      "cn-beijing",
				AccessKey:   "VOLCsigcaptureAKID",
				SecretKey:   "VOLCsigcaptureSECRET1234567890abcdefg",
				Timestamp:   ts,
			},
			expected: Signature{
				Authorization:    "HMAC-SHA256 Credential=VOLCsigcaptureAKID/20260419/cn-beijing/ecs/request, SignedHeaders=content-type;host;x-content-sha256;x-date, Signature=bf85578b2aecf1596c91212dea3e6b11182a683f38cef4c3859b7f7def5fd89f",
				SignedHeaders:    "content-type;host;x-content-sha256;x-date",
				CredentialScope:  "20260419/cn-beijing/ecs/request",
				CanonicalRequest: "GET\n/\nAction=DescribeInstances&MaxResults=100&NextToken=&Version=2020-04-01\ncontent-type:application/x-www-form-urlencoded; charset=utf-8\nhost:ecs.cn-beijing.volcengineapi.com\nx-content-sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\nx-date:20260419T120000Z\n\ncontent-type;host;x-content-sha256;x-date\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				StringToSign:     "HMAC-SHA256\n20260419T120000Z\n20260419/cn-beijing/ecs/request\n08d887bee4a0c6ab8cd47b0e1e956790e276599ccc02252690f09a90d22f32c0",
				XDate:            "20260419T120000Z",
				XContentSHA256:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		},
	}
}

func mustParseFixtureTime(t *testing.T, value string) time.Time {
	t.Helper()
	ts, err := time.Parse(DateFormat, value)
	if err != nil {
		t.Fatalf("parse fixture time %q: %v", value, err)
	}
	return ts
}

func mustParseSignerQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	values, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("parse query %q: %v", raw, err)
	}
	return values
}

func assertSignatureField(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s mismatch\n got: %s\nwant: %s", field, got, want)
	}
}
