package headless

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/404tk/cloudtoolkit/runner"
)

func writeJSON(v any) int {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return exitConfigError
	}
	if _, err := os.Stdout.Write(append(data, '\n')); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return exitConfigError
	}
	return exitSuccess
}

func writeVersion(jsonOutput bool) int {
	if jsonOutput {
		return writeJSON(map[string]string{
			"version": runner.Version(),
		})
	}
	if _, err := fmt.Fprintf(os.Stdout, "CloudToolKit %s\n", runner.Version()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return exitConfigError
	}
	return exitSuccess
}

func fail(jsonOutput bool, code int, err error) int {
	if err == nil {
		return code
	}
	if jsonOutput {
		payload := map[string]any{
			"error":     err.Error(),
			"exit_code": code,
		}
		if coded, ok := err.(codedError); ok {
			payload["code"] = coded.ErrorCode()
		}
		_ = writeJSON(payload)
		return code
	}
	fmt.Fprintln(os.Stderr, err.Error())
	return code
}
