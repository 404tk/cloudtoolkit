package replay

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
)

// gcpDemoServiceAccountJSON is the base64-encoded service account JSON used
// by demo replay for the gcp provider. The RSA key is generated fresh per
// process so the JWT assertion signature sent to the replay token endpoint
// is well-formed (the replay endpoint does not actually validate it).
var gcpDemoServiceAccountJSON = mustBuildGCPServiceAccountJSON()

func mustBuildGCPServiceAccountJSON() string {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return ""
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return ""
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})
	sa := map[string]string{
		"type":           "service_account",
		"project_id":     "ctk-demo-project",
		"private_key_id": "ctkdemo000000000000000000000000000000001",
		"private_key":    string(pemBytes),
		"client_email":   "ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
		"client_id":      "100000000000000000001",
		"token_uri":      "https://oauth2.googleapis.com/token",
	}
	blob, err := json.Marshal(sa)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(blob)
}
