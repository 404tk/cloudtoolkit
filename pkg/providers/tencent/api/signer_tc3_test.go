package api

import (
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

type signerFixture struct {
	Name     string
	Input    signerFixtureInput    `json:"input"`
	Expected signerFixtureExpected `json:"expected"`
}

type signerFixtureInput struct {
	Method      string `json:"method"`
	Service     string `json:"service"`
	Host        string `json:"host"`
	Path        string `json:"path"`
	Query       string `json:"query"`
	Action      string `json:"action"`
	Version     string `json:"version"`
	Region      string `json:"region"`
	Timestamp   int64  `json:"timestamp"`
	SecretID    string `json:"secret_id"`
	SecretKey   string `json:"secret_key"`
	Token       string `json:"token"`
	ContentType string `json:"content_type"`
	Body        string `json:"body"`
}

type signerFixtureExpected struct {
	CanonicalRequest string `json:"canonical_request"`
	StringToSign     string `json:"string_to_sign"`
	Authorization    string `json:"authorization"`
}

func TestTC3SignerFixtures(t *testing.T) {
	if len(signerFixtures) == 0 {
		t.Fatal("no signer fixtures configured")
	}

	signer := TC3Signer{}
	for _, fx := range signerFixtures {
		fx := fx
		t.Run(fx.Name, func(t *testing.T) {
			t.Parallel()

			got, err := signer.Sign(auth.New(fx.Input.SecretID, fx.Input.SecretKey, fx.Input.Token), SignInput{
				Method:      fx.Input.Method,
				Service:     fx.Input.Service,
				Host:        fx.Input.Host,
				Path:        fx.Input.Path,
				Query:       fx.Input.Query,
				ContentType: fx.Input.ContentType,
				Timestamp:   time.Unix(fx.Input.Timestamp, 0).UTC(),
				Payload:     []byte(fx.Input.Body),
			})
			if err != nil {
				t.Fatalf("sign fixture: %v", err)
			}

			if got.Authorization != fx.Expected.Authorization {
				t.Fatalf(
					"authorization mismatch\n  got : %s\n  want: %s\n  canonical mismatch: %t\n  string-to-sign mismatch: %t",
					got.Authorization,
					fx.Expected.Authorization,
					got.CanonicalRequest != fx.Expected.CanonicalRequest,
					got.StringToSign != fx.Expected.StringToSign,
				)
			}
			if got.CanonicalRequest != fx.Expected.CanonicalRequest {
				t.Fatalf("canonical request mismatch\n  got : %s\n  want: %s", got.CanonicalRequest, fx.Expected.CanonicalRequest)
			}
			if got.StringToSign != fx.Expected.StringToSign {
				t.Fatalf("string to sign mismatch\n  got : %s\n  want: %s", got.StringToSign, fx.Expected.StringToSign)
			}
		})
	}
}

func TestTC3SignerSupportsUnsignedPayload(t *testing.T) {
	signer := TC3Signer{}
	got, err := signer.Sign(auth.New("AKIDEXAMPLE", "secretExampleKey", ""), SignInput{
		Method:          "POST",
		Service:         "sts",
		Host:            "sts.tencentcloudapi.com",
		Path:            "/",
		ContentType:     "application/json",
		Timestamp:       time.Unix(1704164645, 0).UTC(),
		Payload:         []byte("{}"),
		UnsignedPayload: true,
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	const want = "438d4109ef0d676b8c2c7ed13cdfcb418e494d53b843d4634ce3b1085f07bb96"
	if got.PayloadHash != want {
		t.Fatalf("unexpected payload hash: %s", got.PayloadHash)
	}
}
