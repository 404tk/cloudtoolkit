package httpclient

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func DecodeJSON(resp *http.Response, body []byte, provider string, out any) error {
	if out == nil || len(body) == 0 || (resp != nil && resp.StatusCode == http.StatusNoContent) {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s response: %w", provider, err)
	}
	return nil
}
