package ufile

import "encoding/json"

// jsonStdUnmarshal is a thin wrapper around encoding/json so the rest of the
// package can rely on a single entry point and tests can override decoding
// behavior if needed.
func jsonStdUnmarshal(body []byte, out any) error {
	return json.Unmarshal(body, out)
}
