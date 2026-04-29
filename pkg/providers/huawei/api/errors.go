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
	return "huawei api error: " + strings.Join(parts, " ")
}

type legacyErrorBody struct {
	Code    string `json:"error_code"`
	Message string `json:"error_msg"`
}

type keystoneErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Title   string `json:"title"`
	} `json:"error"`
}

func DecodeError(statusCode int, body []byte) error {
	if statusCode < http.StatusBadRequest {
		return nil
	}

	var legacy legacyErrorBody
	if err := json.Unmarshal(body, &legacy); err == nil {
		code := strings.TrimSpace(legacy.Code)
		message := strings.TrimSpace(legacy.Message)
		if code != "" || message != "" {
			return &APIError{
				StatusCode: statusCode,
				Code:       code,
				Message:    message,
			}
		}
	}

	var keystone keystoneErrorBody
	if err := json.Unmarshal(body, &keystone); err == nil {
		code := strings.TrimSpace(keystone.Error.Code)
		message := strings.TrimSpace(keystone.Error.Message)
		title := strings.TrimSpace(keystone.Error.Title)
		if code != "" || message != "" || title != "" {
			if message == "" {
				message = title
			}
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
	return fmt.Errorf("huawei api error: status=%d body=%s", statusCode, message)
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

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	code := strings.TrimSpace(apiErr.Code)
	return apiErr.StatusCode == http.StatusNotFound ||
		strings.HasSuffix(code, "ItemNotExist") ||
		strings.HasSuffix(code, ".NotFound")
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
	if strings.Contains(code, "accessdenied") || strings.Contains(code, "forbidden") || strings.Contains(code, "unauthorized") || strings.Contains(code, "nopermission") {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(apiErr.Message))
	return strings.Contains(message, "not authorized") || strings.Contains(message, "access denied") || strings.Contains(message, "forbidden") || strings.Contains(message, "permission")
}

func withRequestID(err error, requestID string) error {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return err
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		copied := *apiErr
		if copied.RequestID == "" {
			copied.RequestID = requestID
		}
		return &copied
	}
	return err
}
