package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type APIErrorBody struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	HTTPStatus int
	Code       int
	Status     string
	Message    string
	RequestID  string
	Service    string
	Action     string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if e.Code != 0 {
		parts = append(parts, fmt.Sprintf("code=%d", e.Code))
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	if len(parts) == 0 {
		if e.HTTPStatus > 0 {
			parts = append(parts, fmt.Sprintf("status=%d", e.HTTPStatus))
		}
		if strings.TrimSpace(e.Status) != "" {
			parts = append(parts, "error_status="+strings.TrimSpace(e.Status))
		}
	}
	if e.RequestID != "" {
		parts = append(parts, "request_id="+e.RequestID)
	}
	return "jdcloud api error: " + strings.Join(parts, " ")
}

func (e *APIError) IsAuthFailure() bool {
	if e == nil {
		return false
	}
	return e.HTTPStatus == http.StatusUnauthorized || e.Code == http.StatusUnauthorized
}

func DecodeError(statusCode int, body []byte) error {
	type errorEnvelope struct {
		RequestID string        `json:"requestId"`
		Error     *APIErrorBody `json:"error,omitempty"`
	}

	var envelope errorEnvelope
	if len(body) != 0 {
		if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error != nil {
			code := envelope.Error.Code
			status := envelope.Error.Status
			message := strings.TrimSpace(envelope.Error.Message)
			requestID := strings.TrimSpace(envelope.RequestID)
			if code != 0 || status != "" || message != "" || requestID != "" {
				return &APIError{
					HTTPStatus: statusCode,
					Code:       code,
					Status:     status,
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
	}
	return &APIError{
		HTTPStatus: statusCode,
		Message:    message,
	}
}

func ErrorCode(err error) int {
	if err == nil {
		return 0
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return 0
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

func (b *APIErrorBody) UnmarshalJSON(data []byte) error {
	type rawBody struct {
		Status  json.RawMessage `json:"status"`
		Code    int             `json:"code"`
		Message string          `json:"message"`
	}
	var raw rawBody
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	status, err := decodeStatus(raw.Status)
	if err != nil {
		return err
	}
	b.Status = status
	b.Code = raw.Code
	b.Message = raw.Message
	return nil
}

func decodeStatus(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", nil
	}

	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		return strings.TrimSpace(text), nil
	}

	var number int
	if err := json.Unmarshal(trimmed, &number); err == nil {
		return strconv.Itoa(number), nil
	}

	return "", fmt.Errorf("jdcloud api error: unsupported status field %s", string(trimmed))
}
