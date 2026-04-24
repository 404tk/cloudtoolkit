package api

import (
	"net/url"
	"reflect"
	"testing"
)

var signerFixtures = []capturedFixture{
	{
		Name: "uaccount_get_user_info_sts",
		Input: fixtureInput{
			AccessKey:     "UCLOUDsigcaptureAKID",
			SecretKey:     "UCLOUDsigcaptureSECRET1234567890abcdefg",
			SecurityToken: "UCLOUDsigcaptureTOKEN789",
			Action:        "GetUserInfo",
		},
		Expected: fixtureExpected{
			ContentType:              "application/x-www-form-urlencoded",
			FormBody:                 `Action=GetUserInfo&PublicKey=UCLOUDsigcaptureAKID&SecurityToken=UCLOUDsigcaptureTOKEN789&Signature=d045a795d4ebaf0db1f987d68ed25727ede89018`,
			FormBodyWithoutSignature: `Action=GetUserInfo&PublicKey=UCLOUDsigcaptureAKID&SecurityToken=UCLOUDsigcaptureTOKEN789`,
			SortedKeys: []string{
				"Action",
				"PublicKey",
				"SecurityToken",
			},
			SigningPayload: `ActionGetUserInfoPublicKeyUCLOUDsigcaptureAKIDSecurityTokenUCLOUDsigcaptureTOKEN789`,
			StringToSign:   `ActionGetUserInfoPublicKeyUCLOUDsigcaptureAKIDSecurityTokenUCLOUDsigcaptureTOKEN789UCLOUDsigcaptureSECRET1234567890abcdefg`,
			Signature:      "d045a795d4ebaf0db1f987d68ed25727ede89018",
		},
	},
	{
		Name: "uhost_describe_instance_basic",
		Input: fixtureInput{
			AccessKey: "UCLOUDsigcaptureAKID",
			SecretKey: "UCLOUDsigcaptureSECRET1234567890abcdefg",
			Action:    "DescribeUHostInstance",
			Region:    "cn-bj2",
			ProjectID: "org-test",
			Params: map[string]any{
				"Limit":  100,
				"Offset": 0,
			},
		},
		Expected: fixtureExpected{
			ContentType:              "application/x-www-form-urlencoded",
			FormBody:                 `Action=DescribeUHostInstance&Limit=100&Offset=0&ProjectId=org-test&PublicKey=UCLOUDsigcaptureAKID&Region=cn-bj2&Signature=38c8845e312235b4dfa77ff608fb3f909a106f90`,
			FormBodyWithoutSignature: `Action=DescribeUHostInstance&Limit=100&Offset=0&ProjectId=org-test&PublicKey=UCLOUDsigcaptureAKID&Region=cn-bj2`,
			SortedKeys: []string{
				"Action",
				"Limit",
				"Offset",
				"ProjectId",
				"PublicKey",
				"Region",
			},
			SigningPayload: `ActionDescribeUHostInstanceLimit100Offset0ProjectIdorg-testPublicKeyUCLOUDsigcaptureAKIDRegioncn-bj2`,
			StringToSign:   `ActionDescribeUHostInstanceLimit100Offset0ProjectIdorg-testPublicKeyUCLOUDsigcaptureAKIDRegioncn-bj2UCLOUDsigcaptureSECRET1234567890abcdefg`,
			Signature:      "38c8845e312235b4dfa77ff608fb3f909a106f90",
		},
	},
	{
		Name: "generic_allocate_backend_batch_long_array",
		Input: fixtureInput{
			AccessKey: "UCLOUDsigcaptureAKID",
			SecretKey: "UCLOUDsigcaptureSECRET1234567890abcdefg",
			Action:    "AllocateBackendBatch",
			Params: map[string]any{
				"Backends": []string{
					"foo",
					"bar",
					"42",
					"foo",
					"bar",
					"42",
					"foo",
					"bar",
					"42",
					"foo",
					"bar",
					"42",
				},
			},
		},
		Expected: fixtureExpected{
			ContentType:              "application/x-www-form-urlencoded",
			FormBody:                 `Action=AllocateBackendBatch&Backends.0=foo&Backends.1=bar&Backends.10=bar&Backends.11=42&Backends.2=42&Backends.3=foo&Backends.4=bar&Backends.5=42&Backends.6=foo&Backends.7=bar&Backends.8=42&Backends.9=foo&PublicKey=UCLOUDsigcaptureAKID&Signature=a905bf1134b2407d40dd00dc76cfa956072bf1e7`,
			FormBodyWithoutSignature: `Action=AllocateBackendBatch&Backends.0=foo&Backends.1=bar&Backends.10=bar&Backends.11=42&Backends.2=42&Backends.3=foo&Backends.4=bar&Backends.5=42&Backends.6=foo&Backends.7=bar&Backends.8=42&Backends.9=foo&PublicKey=UCLOUDsigcaptureAKID`,
			SortedKeys: []string{
				"Action",
				"Backends.0",
				"Backends.1",
				"Backends.10",
				"Backends.11",
				"Backends.2",
				"Backends.3",
				"Backends.4",
				"Backends.5",
				"Backends.6",
				"Backends.7",
				"Backends.8",
				"Backends.9",
				"PublicKey",
			},
			SigningPayload: `ActionAllocateBackendBatchBackends.0fooBackends.1barBackends.10barBackends.1142Backends.242Backends.3fooBackends.4barBackends.542Backends.6fooBackends.7barBackends.842Backends.9fooPublicKeyUCLOUDsigcaptureAKID`,
			StringToSign:   `ActionAllocateBackendBatchBackends.0fooBackends.1barBackends.10barBackends.1142Backends.242Backends.3fooBackends.4barBackends.542Backends.6fooBackends.7barBackends.842Backends.9fooPublicKeyUCLOUDsigcaptureAKIDUCLOUDsigcaptureSECRET1234567890abcdefg`,
			Signature:      "a905bf1134b2407d40dd00dc76cfa956072bf1e7",
		},
	},
}

func TestEncodeFormAndSignatureMatchFixtures(t *testing.T) {
	for _, fx := range loadSignerFixtures(t) {
		fx := fx
		t.Run(fx.Name, func(t *testing.T) {
			form, err := encodeForm(fx.Input.Params)
			if err != nil {
				t.Fatalf("encodeForm() error = %v", err)
			}
			form["Action"] = fx.Input.Action
			if fx.Input.Region != "" {
				form["Region"] = fx.Input.Region
			}
			if fx.Input.ProjectID != "" {
				form["ProjectId"] = fx.Input.ProjectID
			}
			form["PublicKey"] = fx.Input.AccessKey
			if fx.Input.SecurityToken != "" {
				form["SecurityToken"] = fx.Input.SecurityToken
			}

			if got := sortedParamKeys(form); !reflect.DeepEqual(got, fx.Expected.SortedKeys) {
				t.Fatalf("sortedParamKeys() = %v, want %v", got, fx.Expected.SortedKeys)
			}
			if got := signingPayload(form); got != fx.Expected.SigningPayload {
				t.Fatalf("signingPayload() = %q, want %q", got, fx.Expected.SigningPayload)
			}
			if got := signingPayload(form) + fx.Input.SecretKey; got != fx.Expected.StringToSign {
				t.Fatalf("stringToSign = %q, want %q", got, fx.Expected.StringToSign)
			}
			if got := signature(form, fx.Input.SecretKey); got != fx.Expected.Signature {
				t.Fatalf("signature() = %q, want %q", got, fx.Expected.Signature)
			}

			values := url.Values{}
			for key, value := range form {
				values.Set(key, value)
			}
			if got := values.Encode(); got != fx.Expected.FormBodyWithoutSignature {
				t.Fatalf("unsigned body = %q, want %q", got, fx.Expected.FormBodyWithoutSignature)
			}
			values.Set("Signature", fx.Expected.Signature)
			if got := values.Encode(); got != fx.Expected.FormBody {
				t.Fatalf("signed body = %q, want %q", got, fx.Expected.FormBody)
			}
		})
	}
}
