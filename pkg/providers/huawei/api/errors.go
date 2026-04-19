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
	if err := json.Unmarshal(body, &legacy); err == nil &&
		(strings.TrimSpace(legacy.Code) != "" || strings.TrimSpace(legacy.Message) != "") {
		return &APIError{
			StatusCode: statusCode,
			Code:       strings.TrimSpace(legacy.Code),
			Message:    strings.TrimSpace(legacy.Message),
		}
	}

	var keystone keystoneErrorBody
	if err := json.Unmarshal(body, &keystone); err == nil &&
		(strings.TrimSpace(keystone.Error.Code) != "" || strings.TrimSpace(keystone.Error.Message) != "" || strings.TrimSpace(keystone.Error.Title) != "") {
		message := strings.TrimSpace(keystone.Error.Message)
		if message == "" {
			message = strings.TrimSpace(keystone.Error.Title)
		}
		return &APIError{
			StatusCode: statusCode,
			Code:       strings.TrimSpace(keystone.Error.Code),
			Message:    message,
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
