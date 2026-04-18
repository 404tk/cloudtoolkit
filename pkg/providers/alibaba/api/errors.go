package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type APIError struct {
	Code       string
	Message    string
	RequestID  string
	Recommend  string
	HostID     string
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
		parts = append(parts, "alibaba api error")
	}
	msg := strings.Join(parts, ": ")
	if e.RequestID != "" {
		msg += " (request id: " + e.RequestID + ")"
	}
	if e.Recommend != "" {
		msg += " [recommend: " + e.Recommend + "]"
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

type errorEnvelope struct {
	Code      string `json:"Code"`
	Message   string `json:"Message"`
	RequestID string `json:"RequestId"`
	Recommend string `json:"Recommend"`
	HostID    string `json:"HostId"`
	Success   *bool  `json:"Success"`
}

func DecodeError(statusCode int, body []byte) error {
	var envelope errorEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil {
		switch {
		case statusCode >= http.StatusBadRequest && (envelope.Code != "" || envelope.Message != ""):
			return &APIError{
				Code:       envelope.Code,
				Message:    fallbackMessage(envelope.Message, body),
				RequestID:  envelope.RequestID,
				Recommend:  envelope.Recommend,
				HostID:     envelope.HostID,
				StatusCode: statusCode,
			}
		case envelope.Success != nil && !*envelope.Success:
			return &APIError{
				Code:       envelope.Code,
				Message:    fallbackMessage(envelope.Message, body),
				RequestID:  envelope.RequestID,
				Recommend:  envelope.Recommend,
				HostID:     envelope.HostID,
				StatusCode: statusCode,
			}
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

func fallbackMessage(message string, body []byte) string {
	if strings.TrimSpace(message) != "" {
		return message
	}
	return bodySnippet(body)
}

func bodySnippet(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) > 256 {
		return trimmed[:256] + "..."
	}
	return trimmed
}
