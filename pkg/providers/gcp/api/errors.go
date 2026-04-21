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
	Code       int
	Status     string
	Message    string
	Reason     string
	Domain     string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("status=%d", e.StatusCode)}
	if strings.TrimSpace(e.Status) != "" {
		parts = append(parts, "code="+strings.TrimSpace(e.Status))
	}
	if strings.TrimSpace(e.Reason) != "" {
		parts = append(parts, "reason="+strings.TrimSpace(e.Reason))
	}
	message := strings.TrimSpace(e.Message)
	if message == "" {
		return "gcp api error: " + strings.Join(parts, " ")
	}
	return "gcp api error: " + strings.Join(parts, " ") + ": " + message
}

func DecodeError(statusCode int, body []byte) error {
	if statusCode < http.StatusBadRequest {
		return nil
	}

	var payload struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
			Errors  []struct {
				Message string `json:"message"`
				Domain  string `json:"domain"`
				Reason  string `json:"reason"`
			} `json:"errors"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		apiErr := &APIError{
			StatusCode: statusCode,
			Code:       payload.Error.Code,
			Status:     strings.TrimSpace(payload.Error.Status),
			Message:    strings.TrimSpace(payload.Error.Message),
		}
		if len(payload.Error.Errors) > 0 {
			apiErr.Reason = strings.TrimSpace(payload.Error.Errors[0].Reason)
			apiErr.Domain = strings.TrimSpace(payload.Error.Errors[0].Domain)
			if apiErr.Message == "" {
				apiErr.Message = strings.TrimSpace(payload.Error.Errors[0].Message)
			}
		}
		if apiErr.Code != 0 || apiErr.Status != "" || apiErr.Message != "" || apiErr.Reason != "" || apiErr.Domain != "" {
			return apiErr
		}
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return fmt.Errorf("gcp api error: status=%d body=%s", statusCode, message)
}

func IsNotFound(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound || strings.EqualFold(apiErr.Status, "NOT_FOUND")
}

func IsAuthFailure(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden {
		return true
	}
	return strings.EqualFold(apiErr.Status, "PERMISSION_DENIED") || strings.EqualFold(apiErr.Status, "UNAUTHENTICATED")
}

func IsRateLimited(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusTooManyRequests || strings.EqualFold(apiErr.Status, "RESOURCE_EXHAUSTED")
}
