package obs

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
	return "huawei obs api error: " + strings.Join(parts, " ")
}

func decodeError(statusCode int, headers http.Header, body []byte) error {
	if statusCode < http.StatusBadRequest {
		return nil
	}

	var resp errorResponse
	if err := xml.Unmarshal(body, &resp); err == nil &&
		(strings.TrimSpace(resp.Code) != "" || strings.TrimSpace(resp.Message) != "" || strings.TrimSpace(resp.RequestID) != "") {
		return &APIError{
			StatusCode: statusCode,
			Code:       strings.TrimSpace(resp.Code),
			Message:    strings.TrimSpace(resp.Message),
			RequestID:  firstNonEmpty(strings.TrimSpace(resp.RequestID), requestIDFromHeaders(headers)),
		}
	}

	requestID := requestIDFromHeaders(headers)
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}
	if requestID != "" {
		return fmt.Errorf("huawei obs api error: status=%d body=%s request_id=%s", statusCode, message, requestID)
	}
	return fmt.Errorf("huawei obs api error: status=%d body=%s", statusCode, message)
}

func requestIDFromHeaders(headers http.Header) string {
	for key, values := range headers {
		if !strings.EqualFold(key, "x-obs-request-id") && !strings.EqualFold(key, "x-amz-request-id") {
			continue
		}
		if len(values) == 0 {
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
