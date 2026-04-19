package cos

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
)

type APIError struct {
	StatusCode int
	Method     string
	URL        string
	Code       string
	Message    string
	RequestID  string
	TraceID    string
}

func (e *APIError) Error() string {
	parts := make([]string, 0, 2)
	if e.Code != "" {
		parts = append(parts, e.Code)
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	msg := "tencent cos api error"
	if len(parts) > 0 {
		msg += ": " + strings.Join(parts, " ")
	}
	if e.RequestID != "" {
		msg += " request_id=" + e.RequestID
	}
	if e.TraceID != "" {
		msg += " trace_id=" + e.TraceID
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

func decodeError(resp *http.Response, body []byte) error {
	if resp == nil {
		return fmt.Errorf("tencent cos client: nil response")
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= http.StatusMultipleChoices-1 {
		return nil
	}

	var serviceErr errorResponse
	if err := xml.Unmarshal(body, &serviceErr); err == nil {
		method := ""
		requestURL := ""
		if resp.Request != nil {
			method = resp.Request.Method
			if resp.Request.URL != nil {
				requestURL = resp.Request.URL.String()
			}
		}
		requestID := strings.TrimSpace(serviceErr.RequestID)
		if requestID == "" {
			requestID = headerValueIgnoreCase(resp.Header, "X-Cos-Request-Id")
		}
		traceID := strings.TrimSpace(serviceErr.TraceID)
		if traceID == "" {
			traceID = headerValueIgnoreCase(resp.Header, "X-Cos-Trace-Id")
		}
		if serviceErr.Code != "" || serviceErr.Message != "" || requestID != "" || traceID != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				Method:     method,
				URL:        requestURL,
				Code:       strings.TrimSpace(serviceErr.Code),
				Message:    strings.TrimSpace(serviceErr.Message),
				RequestID:  requestID,
				TraceID:    traceID,
			}
		}
	}

	return &HTTPStatusError{
		StatusCode: resp.StatusCode,
		Status:     fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
		Body:       bodySnippet(body),
	}
}

func bodySnippet(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) > 256 {
		return trimmed[:256] + "..."
	}
	return trimmed
}

func headerValueIgnoreCase(headers http.Header, name string) string {
	for key, values := range headers {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return strings.TrimSpace(values[0])
		}
	}
	return ""
}
