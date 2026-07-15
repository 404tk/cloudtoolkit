package headless

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/runner/payloads"
)

func TestExitCodeContract(t *testing.T) {
	contract := map[string]int{
		"success":           ExitSuccess,
		"partial":           ExitPartial,
		"approval_required": ExitApprovalRequired,
		"config_error":      ExitConfigError,
		"unsupported":       ExitUnsupported,
		"execution_error":   ExitExecutionError,
		"deadline_exceeded": ExitDeadlineExceeded,
		"canceled":          ExitCanceled,
	}
	want := map[string]int{
		"success":           0,
		"partial":           2,
		"approval_required": 3,
		"config_error":      4,
		"unsupported":       5,
		"execution_error":   6,
		"deadline_exceeded": 124,
		"canceled":          130,
	}
	for name, value := range contract {
		if value != want[name] {
			t.Fatalf("%s exit code changed: got %d, want %d", name, value, want[name])
		}
	}
}

func TestRunTimeoutPreservesEarlierParentDeadline(t *testing.T) {
	parentDeadline := time.Now().Add(time.Minute)
	parent, parentCancel := context.WithDeadline(context.Background(), parentDeadline)
	defer parentCancel()

	ctx, cancel := withRunTimeout(parent, 10*time.Minute)
	defer cancel()
	got, ok := ctx.Deadline()
	if !ok {
		t.Fatal("run context has no deadline")
	}
	if got.After(parentDeadline) {
		t.Fatalf("run deadline %s exceeds parent deadline %s", got, parentDeadline)
	}
}

func TestErrorCodeExitMapping(t *testing.T) {
	tests := map[payloads.ErrorCode]int{
		payloads.CodeOK:               ExitSuccess,
		payloads.CodePartialFailure:   ExitPartial,
		payloads.CodeApprovalRequired: ExitApprovalRequired,
		payloads.CodeApprovalRejected: ExitApprovalRequired,
		payloads.CodeInvalidArgument:  ExitConfigError,
		payloads.CodeUnsupported:      ExitUnsupported,
		payloads.CodeExecutionFailed:  ExitExecutionError,
		payloads.CodeOutputFailed:     ExitExecutionError,
		payloads.CodeDeadlineExceeded: ExitDeadlineExceeded,
		payloads.CodeCanceled:         ExitCanceled,
	}
	for code, want := range tests {
		if got := exitCodeFor(code); got != want {
			t.Errorf("exitCodeFor(%q) = %d, want %d", code, got, want)
		}
	}
}

func TestErrorCodeContract(t *testing.T) {
	contract := map[payloads.ErrorCode]string{
		payloads.CodeOK:               "ok",
		payloads.CodeInvalidArgument:  "invalid_argument",
		payloads.CodePartialFailure:   "partial_failure",
		payloads.CodeApprovalRequired: "approval_required",
		payloads.CodeApprovalRejected: "approval_rejected",
		payloads.CodeUnsupported:      "unsupported",
		payloads.CodeExecutionFailed:  "execution_failed",
		payloads.CodeOutputFailed:     "output_failed",
		payloads.CodeDeadlineExceeded: "deadline_exceeded",
		payloads.CodeCanceled:         "canceled",
	}
	for code, want := range contract {
		if string(code) != want {
			t.Errorf("error code changed: got %q, want %q", code, want)
		}
	}
}

func TestResultEnvelopeContract(t *testing.T) {
	data, err := json.Marshal(resultEnvelope{
		SchemaVersion: resultSchemaVersion,
		Status:        payloads.ResultFailure,
		Code:          payloads.CodeExecutionFailed,
		ExitCode:      ExitExecutionError,
		Result:        map[string]string{"provider": "replay"},
		Error:         "remote operation failed",
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"schema_version", "status", "code", "exit_code", "result", "error"} {
		if _, ok := got[field]; !ok {
			t.Errorf("result envelope missing %q", field)
		}
	}
}
