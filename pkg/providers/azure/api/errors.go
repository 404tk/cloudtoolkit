package api

import (
	"encoding/json"
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
	parts := make([]string, 0, 3)
	if strings.TrimSpace(e.Code) != "" {
		parts = append(parts, strings.TrimSpace(e.Code))
	}
	if strings.TrimSpace(e.Message) != "" {
		parts = append(parts, strings.TrimSpace(e.Message))
	}
	if len(parts) == 0 {
		parts = append(parts, http.StatusText(e.StatusCode))
	}
	if strings.TrimSpace(e.RequestID) != "" {
		parts = append(parts, "request_id="+strings.TrimSpace(e.RequestID))
	}
	return "azure api error: " + strings.Join(parts, " ")
}

func DecodeError(statusCode int, body []byte) error {
	if statusCode < http.StatusBadRequest {
		return nil
	}

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		code := strings.TrimSpace(payload.Error.Code)
		message := strings.TrimSpace(payload.Error.Message)
		if code != "" || message != "" {
			return &APIError{
				StatusCode: statusCode,
				Code:       code,
				Message:    message,
			}
		}
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return fmt.Errorf("azure api error: status=%d body=%s", statusCode, message)
}

func IsNotFound(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode == http.StatusNotFound {
		return true
	}
	code := strings.TrimSpace(strings.ToLower(apiErr.Code))
	return code == "notfound" || code == "resourcenotfound"
}

func IsAuthFailure(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden {
			return true
		}
		if strings.Contains(strings.ToUpper(apiErr.Code), "AADSTS") {
			return true
		}
		if strings.Contains(strings.ToUpper(apiErr.Message), "AADSTS") {
			return true
		}
	}
	return strings.Contains(strings.ToUpper(err.Error()), "AADSTS")
}

func withRequestID(err error, requestID string) error {
	if err == nil {
		return nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && strings.TrimSpace(apiErr.RequestID) == "" {
		apiErr.RequestID = strings.TrimSpace(requestID)
	}
	return err
}
