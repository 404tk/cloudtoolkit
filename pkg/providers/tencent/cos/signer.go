package cos

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

const (
	sha1SignAlgorithm   = "sha1"
	privateHeaderPrefix = "x-cos-"
	privateCIHeaderPref = "x-ci-"
	defaultAuthExpire   = time.Hour
)

var needSignHeaders = map[string]bool{
	"host":                           true,
	"range":                          true,
	"x-cos-acl":                      true,
	"x-cos-grant-read":               true,
	"x-cos-grant-write":              true,
	"x-cos-grant-full-control":       true,
	"cache-control":                  true,
	"content-disposition":            true,
	"content-encoding":               true,
	"content-type":                   true,
	"content-length":                 true,
	"content-md5":                    true,
	"transfer-encoding":              true,
	"expect":                         true,
	"expires":                        true,
	"x-cos-content-sha1":             true,
	"x-cos-storage-class":            true,
	"if-match":                       true,
	"if-modified-since":              true,
	"if-none-match":                  true,
	"if-unmodified-since":            true,
	"origin":                         true,
	"access-control-request-method":  true,
	"access-control-request-headers": true,
	"x-cos-object-type":              true,
	"pic-operations":                 true,
}

type authTime struct {
	SignStartTime time.Time
	SignEndTime   time.Time
	KeyStartTime  time.Time
	KeyEndTime    time.Time
}

func Sign(req *http.Request, cred auth.Credential, now time.Time) error {
	if req == nil {
		return fmt.Errorf("tencent cos signer: nil request")
	}
	if req.URL == nil {
		return fmt.Errorf("tencent cos signer: nil request url")
	}
	if err := cred.Validate(); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	window := &authTime{
		SignStartTime: now,
		SignEndTime:   now.Add(defaultAuthExpire),
		KeyStartTime:  now,
		KeyEndTime:    now.Add(defaultAuthExpire),
	}
	return addAuthorizationHeader(req, cred, window)
}

func addAuthorizationHeader(req *http.Request, cred auth.Credential, window *authTime) error {
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	if cred.Token != "" {
		req.Header.Set("x-cos-security-token", cred.Token)
	}
	authorization, err := buildAuthorization(cred.SecretID, cred.SecretKey, req, window, true)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorization)
	return nil
}

func buildAuthorization(secretID, secretKey string, req *http.Request, window *authTime, signHost bool) (string, error) {
	if req == nil || req.URL == nil {
		return "", fmt.Errorf("tencent cos signer: nil request")
	}
	if window == nil {
		return "", fmt.Errorf("tencent cos signer: nil auth window")
	}
	signTime := window.signString()
	keyTime := window.keyString()
	signKey := calSignKey(secretKey, keyTime)

	if signHost {
		host := strings.TrimSpace(req.Host)
		if host == "" {
			host = req.URL.Host
		}
		if host == "" {
			return "", fmt.Errorf("tencent cos signer: empty host")
		}
		req.Host = host
		req.Header.Set("Host", host)
	}

	formatHeaders, signedHeaderList := genFormatHeaders(req.Header)
	formatParameters, signedParameterList := genFormatParameters(req.URL.Query())
	formatString := genFormatString(req.Method, *req.URL, formatParameters, formatHeaders)
	stringToSign := calStringToSign(sha1SignAlgorithm, keyTime, formatString)
	signature := calSignature(signKey, stringToSign)

	return genAuthorization(secretID, signTime, keyTime, signature, signedHeaderList, signedParameterList), nil
}

func (a *authTime) signString() string {
	return fmt.Sprintf("%d;%d", a.SignStartTime.Unix(), a.SignEndTime.Unix())
}

func (a *authTime) keyString() string {
	return fmt.Sprintf("%d;%d", a.KeyStartTime.Unix(), a.KeyEndTime.Unix())
}

type valuesSignMap map[string][]string

func (vs valuesSignMap) Add(key, value string) {
	key = strings.ToLower(safeURLEncode(key))
	vs[key] = append(vs[key], value)
}

func (vs valuesSignMap) Encode() string {
	var keys []string
	for k := range vs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		items := vs[k]
		sort.Strings(items)
		for _, val := range items {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, safeURLEncode(val)))
		}
	}
	return strings.Join(pairs, "&")
}

func safeURLEncode(s string) string {
	s = encodeURIComponent(s)
	s = strings.ReplaceAll(s, "!", "%21")
	s = strings.ReplaceAll(s, "'", "%27")
	s = strings.ReplaceAll(s, "(", "%28")
	s = strings.ReplaceAll(s, ")", "%29")
	s = strings.ReplaceAll(s, "*", "%2A")
	return s
}

func genFormatString(method string, uri url.URL, formatParameters, formatHeaders string) string {
	return fmt.Sprintf("%s\n%s\n%s\n%s\n", strings.ToLower(method), uri.Path, formatParameters, formatHeaders)
}

func genFormatParameters(parameters url.Values) (formatParameters string, signedParameterList []string) {
	ps := valuesSignMap{}
	for key, values := range parameters {
		for _, value := range values {
			ps.Add(key, value)
			signedParameterList = append(signedParameterList, strings.ToLower(safeURLEncode(key)))
		}
	}
	formatParameters = ps.Encode()
	sort.Strings(signedParameterList)
	return
}

func genFormatHeaders(headers http.Header) (formatHeaders string, signedHeaderList []string) {
	hs := valuesSignMap{}
	for key, values := range headers {
		if isSignHeader(strings.ToLower(key)) {
			for _, value := range values {
				hs.Add(key, value)
				signedHeaderList = append(signedHeaderList, strings.ToLower(safeURLEncode(key)))
			}
		}
	}
	formatHeaders = hs.Encode()
	sort.Strings(signedHeaderList)
	return
}

func calSignKey(secretKey, keyTime string) string {
	digest := calHMACDigest(secretKey, keyTime, sha1SignAlgorithm)
	return fmt.Sprintf("%x", digest)
}

func calStringToSign(signAlgorithm, signTime, formatString string) string {
	h := sha1.New()
	_, _ = h.Write([]byte(formatString))
	return fmt.Sprintf("%s\n%s\n%x\n", signAlgorithm, signTime, h.Sum(nil))
}

func calSignature(signKey, stringToSign string) string {
	digest := calHMACDigest(signKey, stringToSign, sha1SignAlgorithm)
	return fmt.Sprintf("%x", digest)
}

func genAuthorization(secretID, signTime, keyTime, signature string, signedHeaderList, signedParameterList []string) string {
	return strings.Join([]string{
		"q-sign-algorithm=" + sha1SignAlgorithm,
		"q-ak=" + secretID,
		"q-sign-time=" + signTime,
		"q-key-time=" + keyTime,
		"q-header-list=" + strings.Join(signedHeaderList, ";"),
		"q-url-param-list=" + strings.Join(signedParameterList, ";"),
		"q-signature=" + signature,
	}, "&")
}

func calHMACDigest(key, msg, signMethod string) []byte {
	var hashFunc func() hash.Hash
	switch signMethod {
	case "sha1":
		hashFunc = sha1.New
	default:
		hashFunc = sha1.New
	}
	h := hmac.New(hashFunc, []byte(key))
	_, _ = h.Write([]byte(msg))
	return h.Sum(nil)
}

func isSignHeader(key string) bool {
	if needSignHeaders[key] {
		return true
	}
	if strings.HasPrefix(key, privateCIHeaderPref) {
		return true
	}
	return strings.HasPrefix(key, privateHeaderPrefix)
}

func encodeURIComponent(s string, excluded ...[]byte) string {
	var b bytes.Buffer
	written := 0

	for i, n := 0, len(s); i < n; i++ {
		c := s[i]

		switch c {
		case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
			continue
		default:
			if 'a' <= c && c <= 'z' {
				continue
			}
			if 'A' <= c && c <= 'Z' {
				continue
			}
			if '0' <= c && c <= '9' {
				continue
			}
			if len(excluded) > 0 {
				skip := false
				for _, ch := range excluded[0] {
					if ch == c {
						skip = true
						break
					}
				}
				if skip {
					continue
				}
			}
		}

		b.WriteString(s[written:i])
		fmt.Fprintf(&b, "%%%02X", c)
		written = i + 1
	}

	if written == 0 {
		return s
	}
	b.WriteString(s[written:])
	return b.String()
}
