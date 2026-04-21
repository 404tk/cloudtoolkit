package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
)

func ParseRSAPrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(pemData)))
	if block == nil {
		return nil, errors.New("gcp: invalid private key pem")
	}

	switch block.Type {
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("gcp: service account private key is not RSA")
		}
		return rsaKey, nil
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("gcp: unsupported private key type %q", block.Type)
	}
}

func SignAssertion(cred Credential, now time.Time) (string, error) {
	if err := cred.Validate(); err != nil {
		return "", err
	}

	key, err := ParseRSAPrivateKey(cred.PrivateKeyPEM)
	if err != nil {
		return "", err
	}

	type header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
		Kid string `json:"kid,omitempty"`
	}
	type claims struct {
		Iss   string `json:"iss"`
		Scope string `json:"scope"`
		Aud   string `json:"aud"`
		Iat   int64  `json:"iat"`
		Exp   int64  `json:"exp"`
	}

	headerJSON, err := json.Marshal(header{
		Alg: "RS256",
		Typ: "JWT",
		Kid: strings.TrimSpace(cred.PrivateKeyID),
	})
	if err != nil {
		return "", err
	}

	claimsJSON, err := json.Marshal(claims{
		Iss:   cred.ClientEmail,
		Scope: strings.Join(cred.Scopes, " "),
		Aud:   cred.TokenURI,
		Iat:   now.Unix(),
		Exp:   now.Add(time.Hour).Unix(),
	})
	if err != nil {
		return "", err
	}

	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	sum := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}
