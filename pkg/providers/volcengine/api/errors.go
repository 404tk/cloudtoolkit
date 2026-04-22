package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type ResponseMetadata struct {
	RequestID string     `json:"RequestId"`
	Error     *ErrorBody `json:"Error,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
	CodeN   int    `json:"CodeN"`
}

type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
	Service    string
	Action     string
	RequestID  string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if e.Code != "" {
		parts = append(parts, e.Code)
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	if len(parts) == 0 && e.HTTPStatus > 0 {
		parts = append(parts, fmt.Sprintf("status=%d", e.HTTPStatus))
	}
	if e.RequestID != "" {
		parts = append(parts, "request_id="+e.RequestID)
	}
	return "volcengine api error: " + strings.Join(parts, " ")
}

func DecodeError(statusCode int, body []byte) error {
	var envelope struct {
		ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	}
	if len(body) != 0 {
		if err := json.Unmarshal(body, &envelope); err == nil && envelope.ResponseMetadata.Error != nil {
			code := strings.TrimSpace(envelope.ResponseMetadata.Error.Code)
			message := strings.TrimSpace(envelope.ResponseMetadata.Error.Message)
			requestID := strings.TrimSpace(envelope.ResponseMetadata.RequestID)
			if code != "" || message != "" || requestID != "" {
				return &APIError{
					HTTPStatus: statusCode,
					Code:       code,
					Message:    message,
					RequestID:  requestID,
				}
			}
		}
	}

	if statusCode < http.StatusBadRequest {
		return nil
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(statusCode)
	} else {
		message = "decoded body: " + message
	}
	return &APIError{
		HTTPStatus: statusCode,
		Message:    message,
	}
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
	if strings.Contains(code, "accessdenied") || strings.Contains(code, "unauthorized") {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(apiErr.Message))
	return strings.Contains(message, "not authorized") || strings.Contains(message, "access denied")
}

func annotateError(err error, service, action string) error {
	if err == nil {
		return nil
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	copied := *apiErr
	copied.Service = strings.TrimSpace(service)
	copied.Action = strings.TrimSpace(action)
	return &copied
}
