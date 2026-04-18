package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

const (
	tc3Algorithm     = "TC3-HMAC-SHA256"
	tc3SignedHeaders = "content-type;host"
)

type SignInput struct {
	Method          string
	Service         string
	Host            string
	Path            string
	Query           string
	ContentType     string
	Timestamp       time.Time
	Payload         []byte
	UnsignedPayload bool
}

type Signature struct {
	Authorization    string
	CanonicalRequest string
	StringToSign     string
	CredentialScope  string
	SignedHeaders    string
	PayloadHash      string
}

type TC3Signer struct{}

func (s TC3Signer) Sign(credential auth.Credential, in SignInput) (Signature, error) {
	if err := credential.Validate(); err != nil {
		return Signature{}, err
	}
	if in.Service == "" {
		return Signature{}, errors.New("tencent signer: empty service")
	}
	if in.Host == "" {
		return Signature{}, errors.New("tencent signer: empty host")
	}

	method := strings.ToUpper(in.Method)
	if method == "" {
		method = http.MethodPost
	}
	path := in.Path
	if path == "" {
		path = "/"
	}
	contentType := in.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	timestamp := in.Timestamp.UTC()
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	payloadHash := sha256Hex(in.Payload)
	if in.UnsignedPayload {
		payloadHash = sha256Hex([]byte("UNSIGNED-PAYLOAD"))
	}
	canonicalHeaders := fmt.Sprintf(
		"content-type:%s\nhost:%s\n",
		contentType,
		in.Host,
	)
	canonicalRequest := fmt.Sprintf(
		"%s\n%s\n%s\n%s\n%s\n%s",
		method,
		path,
		in.Query,
		canonicalHeaders,
		tc3SignedHeaders,
		payloadHash,
	)

	requestUnix := strconv.FormatInt(timestamp.Unix(), 10)
	date := timestamp.Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, in.Service)
	stringToSign := fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		tc3Algorithm,
		requestUnix,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	)

	secretDate := hmacSHA256([]byte("TC3"+credential.SecretKey), date)
	secretService := hmacSHA256(secretDate, in.Service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signatureHex := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))
	authorization := fmt.Sprintf(
		"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		tc3Algorithm,
		credential.SecretID,
		credentialScope,
		tc3SignedHeaders,
		signatureHex,
	)

	return Signature{
		Authorization:    authorization,
		CanonicalRequest: canonicalRequest,
		StringToSign:     stringToSign,
		CredentialScope:  credentialScope,
		SignedHeaders:    tc3SignedHeaders,
		PayloadHash:      payloadHash,
	}, nil
}

func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
