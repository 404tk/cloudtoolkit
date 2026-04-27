package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

type SignInput struct {
	Method string
	Params url.Values
}

type RPCSigner struct{}

func (s RPCSigner) Sign(credential auth.Credential, input SignInput) (url.Values, error) {
	if err := credential.Validate(); err != nil {
		return nil, err
	}
	params := httpclient.CloneValues(input.Params)
	params.Del("Signature")
	stringToSign := buildRPCStringToSign(input.Method, params)
	params.Set("Signature", signHMAC1(stringToSign, credential.AccessKeySecret+"&"))
	return params, nil
}

func buildRPCStringToSign(method string, params url.Values) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}
	formed := params.Encode()
	formed = strings.ReplaceAll(formed, "+", "%20")
	formed = strings.ReplaceAll(formed, "*", "%2A")
	formed = strings.ReplaceAll(formed, "%7E", "~")
	return method + "&%2F&" + url.QueryEscape(formed)
}

func signHMAC1(source, secret string) string {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(source))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func defaultNonce() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "nonce"
	}
	return hex.EncodeToString(buf[:])
}
