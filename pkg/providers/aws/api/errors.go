package api

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	parts := []string{}
	if e.Code != "" {
		parts = append(parts, e.Code)
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	if len(parts) == 0 {
		parts = append(parts, http.StatusText(e.StatusCode))
	}
	if e.RequestID != "" {
		parts = append(parts, "request_id="+e.RequestID)
	}
	return "aws api error: " + strings.Join(parts, " ")
}

type errorResponse struct {
	XMLName   xml.Name  `xml:"ErrorResponse"`
	Error     errorBody `xml:"Error"`
	RequestID string    `xml:"RequestId"`
}

type s3ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	RequestID string   `xml:"RequestId"`
}

type errorBody struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func DecodeError(statusCode int, body []byte) error {
	if statusCode < http.StatusBadRequest {
		return nil
	}

	var resp errorResponse
	if err := xml.NewDecoder(bytes.NewReader(body)).Decode(&resp); err == nil {
		code := strings.TrimSpace(resp.Error.Code)
		message := strings.TrimSpace(resp.Error.Message)
		requestID := strings.TrimSpace(resp.RequestID)
		if code != "" || message != "" || requestID != "" {
			return &APIError{
				StatusCode: statusCode,
				Code:       code,
				Message:    message,
				RequestID:  requestID,
			}
		}
	}

	var s3Resp s3ErrorResponse
	if err := xml.NewDecoder(bytes.NewReader(body)).Decode(&s3Resp); err == nil {
		code := strings.TrimSpace(s3Resp.Code)
		message := strings.TrimSpace(s3Resp.Message)
		requestID := strings.TrimSpace(s3Resp.RequestID)
		if code != "" || message != "" || requestID != "" {
			return &APIError{
				StatusCode: statusCode,
				Code:       code,
				Message:    message,
				RequestID:  requestID,
			}
		}
	}

	// JSON-1.1 envelope (SSM, ECR, and other JSON RPC services). Errors look
	// like `{"__type": "<service>#<Code>", "Message": "..."}`.
	if len(body) > 0 && body[0] == '{' {
		var jsonResp struct {
			Type    string `json:"__type"`
			Code    string `json:"Code"`
			Message string `json:"message"`
			Msg     string `json:"Message"`
		}
		if err := json.Unmarshal(body, &jsonResp); err == nil {
			code := strings.TrimSpace(jsonResp.Code)
			if code == "" && jsonResp.Type != "" {
				code = jsonResp.Type
				if idx := strings.LastIndex(code, "#"); idx >= 0 {
					code = code[idx+1:]
				}
			}
			message := strings.TrimSpace(jsonResp.Message)
			if message == "" {
				message = strings.TrimSpace(jsonResp.Msg)
			}
			if code != "" || message != "" {
				return &APIError{
					StatusCode: statusCode,
					Code:       code,
					Message:    message,
				}
			}
		}
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return fmt.Errorf("aws api error: status=%d message=%s", statusCode, message)
}

func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return strings.TrimSpace(apiErr.Code)
	}
	return ""
}

func IsAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	code := strings.ToLower(strings.TrimSpace(apiErr.Code))
	if strings.Contains(code, "accessdenied") || strings.Contains(code, "unauthorizedoperation") || strings.Contains(code, "unauthorized") {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(apiErr.Message))
	return strings.Contains(message, "not authorized") || strings.Contains(message, "access denied") || strings.Contains(message, "unauthorized")
}
