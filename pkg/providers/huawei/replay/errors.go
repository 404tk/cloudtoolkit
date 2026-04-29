package replay

import (
	"fmt"
	"net/http"
	"strings"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type apiErrorPayload struct {
	ErrorCode string `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
	RequestID string `json:"request_id,omitempty"`
}

func apiErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	payload := apiErrorPayload{
		ErrorCode: strings.TrimSpace(code),
		ErrorMsg:  strings.TrimSpace(message),
		RequestID: "req-replay-" + fmt.Sprintf("%d", statusCode),
	}
	resp := demoreplay.JSONResponse(req, statusCode, payload)
	resp.Header.Set("X-Request-Id", payload.RequestID)
	return resp
}
