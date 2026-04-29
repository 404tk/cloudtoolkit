package replay

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

// AuthFailureKind classifies the outcome of replay-side request authentication.
type AuthFailureKind int

const (
	AuthOK AuthFailureKind = iota
	AuthInvalidAccessKey
	AuthInvalidSignature
)

// SubtleEqual reports whether two strings are equal in constant time.
func SubtleEqual(left, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

// ReadRequestBody drains req.Body and rewires it so downstream handlers can read it again.
func ReadRequestBody(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// JSONResponse builds a 200-style JSON response bound to req with the marshalled payload.
func JSONResponse(req *http.Request, statusCode int, payload any) *http.Response {
	body, _ := json.Marshal(payload)
	return Response(req, statusCode, "application/json", body)
}

// XMLResponse builds a response bound to req with the marshalled XML payload.
func XMLResponse(req *http.Request, statusCode int, payload any) *http.Response {
	body, _ := xml.Marshal(payload)
	return Response(req, statusCode, "application/xml", body)
}

// Response is the shared response constructor used by JSONResponse / XMLResponse.
func Response(req *http.Request, statusCode int, contentType string, body []byte) *http.Response {
	if body == nil {
		body = []byte{}
	}
	return &http.Response{
		StatusCode:    statusCode,
		Status:        fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:        http.Header{"Content-Type": []string{contentType}},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
		ProtoMajor:    1,
		ProtoMinor:    1,
	}
}
