package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type APIError struct {
	Code       string
	Message    string
	RequestID  string
	StatusCode int
}

func (e *APIError) Error() string {
	parts := make([]string, 0, 3)
	if e.Code != "" {
		parts = append(parts, e.Code)
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	if len(parts) == 0 {
		parts = append(parts, "tencent api error")
	}
	msg := strings.Join(parts, ": ")
	if e.RequestID != "" {
		msg += " (request id: " + e.RequestID + ")"
	}
	if e.StatusCode > 0 {
		msg += fmt.Sprintf(" [status=%d]", e.StatusCode)
	}
	return msg
}

type HTTPStatusError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPStatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("unexpected http status: %s", e.Status)
	}
	return fmt.Sprintf("unexpected http status: %s: %s", e.Status, e.Body)
}

type responseEnvelope struct {
	Response struct {
		Error *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

func DecodeError(statusCode int, body []byte) error {
	var envelope responseEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Response.Error != nil {
		return &APIError{
			Code:       envelope.Response.Error.Code,
			Message:    envelope.Response.Error.Message,
			RequestID:  envelope.Response.RequestID,
			StatusCode: statusCode,
		}
	}
	if statusCode >= http.StatusBadRequest {
		return &HTTPStatusError{
			StatusCode: statusCode,
			Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
			Body:       bodySnippet(body),
		}
	}
	return nil
}

func bodySnippet(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) > 256 {
		return trimmed[:256] + "..."
	}
	return trimmed
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
	if strings.Contains(code, "unauthorizedoperation") || strings.Contains(code, "accessdenied") {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(apiErr.Message))
	return strings.Contains(message, "not authorized") || strings.Contains(message, "access denied") || strings.Contains(message, "unauthorized")
}
