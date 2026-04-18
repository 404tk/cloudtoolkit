package api

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

func TestBuildRPCStringToSign(t *testing.T) {
	t.Parallel()

	stringToSign := buildRPCStringToSign("", url.Values{})
	if want := "GET&%2F&"; stringToSign != want {
		t.Fatalf("unexpected empty string-to-sign: got %q want %q", stringToSign, want)
	}

	values := url.Values{}
	values.Set("key", "value")
	stringToSign = buildRPCStringToSign("", values)
	if want := "GET&%2F&key%3Dvalue"; stringToSign != want {
		t.Fatalf("unexpected single param string-to-sign: got %q want %q", stringToSign, want)
	}

	values.Set("q", "value")
	stringToSign = buildRPCStringToSign("", values)
	if want := "GET&%2F&key%3Dvalue%26q%3Dvalue"; stringToSign != want {
		t.Fatalf("unexpected two param string-to-sign: got %q want %q", stringToSign, want)
	}

	values.Set("q", "http://domain/?q=value&q2=value2")
	stringToSign = buildRPCStringToSign("", values)
	if want := "GET&%2F&key%3Dvalue%26q%3Dhttp%253A%252F%252Fdomain%252F%253Fq%253Dvalue%2526q2%253Dvalue2"; stringToSign != want {
		t.Fatalf("unexpected escaped string-to-sign: got %q want %q", stringToSign, want)
	}
}

func TestRPCSignerMatchesOfficialSDKVectors(t *testing.T) {
	t.Parallel()

	signer := RPCSigner{}
	values := url.Values{}
	values.Set("Action", "")
	values.Set("Version", "")
	values.Set("Format", "JSON")
	values.Set("Timestamp", "mock date")
	values.Set("SignatureMethod", "HMAC-SHA1")
	values.Set("SignatureVersion", "1.0")
	values.Set("SignatureType", "")
	values.Set("SignatureNonce", "MOCK_UUID")
	values.Set("AccessKeyId", "accessKeyId")
	values.Set("RegionId", "regionId")

	signed, err := signer.Sign(auth.New("accessKeyId", "accessKeySecret", ""), SignInput{
		Method: http.MethodGet,
		Params: values,
	})
	if err != nil {
		t.Fatalf("sign ak request: %v", err)
	}
	if got, want := signed.Get("Signature"), "7loPmFjvDnzOVnQeQNj85S6nFGY="; got != want {
		t.Fatalf("unexpected ak signature: got %q want %q", got, want)
	}

	values.Set("SecurityToken", "accessKeyStsToken")
	signed, err = signer.Sign(auth.New("accessKeyId", "accessKeySecret", "accessKeyStsToken"), SignInput{
		Method: http.MethodGet,
		Params: values,
	})
	if err != nil {
		t.Fatalf("sign sts request: %v", err)
	}
	if got, want := signed.Get("Signature"), "5Nxdcler+ihqWqv0Hr2On4PsBf4="; got != want {
		t.Fatalf("unexpected sts signature: got %q want %q", got, want)
	}
}
