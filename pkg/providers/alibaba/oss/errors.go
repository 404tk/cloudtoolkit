package oss

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
	HostID     string
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
	if e.RequestID != "" {
		parts = append(parts, "request_id="+e.RequestID)
	}
	if e.HostID != "" {
		parts = append(parts, "host_id="+e.HostID)
	}
	if len(parts) == 0 {
		parts = append(parts, http.StatusText(e.StatusCode))
	}
	return "alibaba oss api error: " + strings.Join(parts, " ")
}

func decodeError(resp *http.Response, body []byte) error {
	if resp == nil {
		return fmt.Errorf("alibaba oss client: nil response")
	}
	if resp.StatusCode < http.StatusBadRequest {
		return nil
	}

	var serviceErr errorResponse
	if err := xml.Unmarshal(body, &serviceErr); err == nil &&
		(strings.TrimSpace(serviceErr.Code) != "" || strings.TrimSpace(serviceErr.Message) != "" || strings.TrimSpace(serviceErr.RequestID) != "") {
		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       strings.TrimSpace(serviceErr.Code),
			Message:    strings.TrimSpace(serviceErr.Message),
			RequestID:  firstNonEmpty(strings.TrimSpace(serviceErr.RequestID), headerValueIgnoreCase(resp.Header, "x-oss-request-id")),
			HostID:     strings.TrimSpace(serviceErr.HostID),
		}
	}

	requestID := headerValueIgnoreCase(resp.Header, "x-oss-request-id")
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	if requestID != "" {
		return fmt.Errorf("alibaba oss api error: status=%d body=%s request_id=%s", resp.StatusCode, message, requestID)
	}
	return fmt.Errorf("alibaba oss api error: status=%d body=%s", resp.StatusCode, message)
}

func headerValueIgnoreCase(headers http.Header, name string) string {
	for key, values := range headers {
		if !strings.EqualFold(key, name) || len(values) == 0 {
			continue
		}
		if value := strings.TrimSpace(values[0]); value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
