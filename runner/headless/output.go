package headless

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/404tk/cloudtoolkit/runner"
	"github.com/404tk/cloudtoolkit/runner/payloads"
)

const resultSchemaVersion = "1"

type resultEnvelope struct {
	SchemaVersion string                `json:"schema_version"`
	Status        payloads.ResultStatus `json:"status"`
	Code          payloads.ErrorCode    `json:"code"`
	ExitCode      int                   `json:"exit_code"`
	Result        any                   `json:"result,omitempty"`
	Error         string                `json:"error,omitempty"`
}

func writeJSON(v any) int {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return exitExecutionError
	}
	if _, err := os.Stdout.Write(append(data, '\n')); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return exitExecutionError
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
		return exitExecutionError
	}
	return exitSuccess
}

func writeResultJSON(result payloads.Result, exitCode int) int {
	envelope := resultEnvelope{
		SchemaVersion: resultSchemaVersion,
		Status:        result.Status,
		Code:          result.Code,
		ExitCode:      exitCode,
		Result:        result.Value,
	}
	if result.Err != nil {
		envelope.Error = result.Err.Error()
	}
	return writeJSON(envelope)
}

func fail(jsonOutput bool, code int, err error) int {
	errorCode := payloads.CodeExecutionFailed
	switch code {
	case exitConfigError:
		errorCode = payloads.CodeInvalidArgument
	case exitUnsupported:
		errorCode = payloads.CodeUnsupported
	case exitApprovalRequired:
		errorCode = payloads.CodeApprovalRequired
	case exitDeadlineExceeded:
		errorCode = payloads.CodeDeadlineExceeded
	case exitCanceled:
		errorCode = payloads.CodeCanceled
	}
	if coded, ok := err.(codedError); ok {
		errorCode = coded.ErrorCode()
	}
	return failWithCode(jsonOutput, code, errorCode, err)
}

func failWithCode(jsonOutput bool, exitCode int, errorCode payloads.ErrorCode, err error) int {
	if err == nil {
		return exitCode
	}
	if jsonOutput {
		if writeCode := writeResultJSON(payloads.Result{
			Status: payloads.ResultFailure,
			Code:   errorCode,
			Err:    err,
		}, exitCode); writeCode != exitSuccess {
			return writeCode
		}
		return exitCode
	}
	fmt.Fprintln(os.Stderr, err.Error())
	return exitCode
}
