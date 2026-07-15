package payloads

import (
	"context"
	"errors"
	"testing"
)

type countingProducer struct {
	calls int
	value any
	err   error
}

func (p *countingProducer) Result(context.Context, map[string]string) (any, error) {
	p.calls++
	return p.value, p.err
}

func TestExecuteInvokesResultOnce(t *testing.T) {
	producer := &countingProducer{value: map[string]string{"status": "success"}}
	result := Execute(context.Background(), nil, producer)
	if producer.calls != 1 {
		t.Fatalf("Result called %d times, want 1", producer.calls)
	}
	if result.Status != ResultSuccess || result.Code != CodeOK || result.Err != nil {
		t.Fatalf("unexpected normalized result: %#v", result)
	}
}

func TestExecutePreservesStructuredFailure(t *testing.T) {
	payload := map[string]string{"status": "error"}
	producer := &countingProducer{err: NewResultError(payload, CodeExecutionFailed, errors.New("failed"))}
	result := Execute(context.Background(), nil, producer)
	if producer.calls != 1 {
		t.Fatalf("Result called %d times, want 1", producer.calls)
	}
	if result.Status != ResultFailure || result.Code != CodeExecutionFailed || result.Value == nil {
		t.Fatalf("unexpected normalized result: %#v", result)
	}
}

func TestExecuteMapsContextErrors(t *testing.T) {
	tests := []struct {
		err  error
		code ErrorCode
	}{
		{context.Canceled, CodeCanceled},
		{context.DeadlineExceeded, CodeDeadlineExceeded},
	}
	for _, test := range tests {
		producer := &countingProducer{err: NewResultError(nil, CodeExecutionFailed, test.err)}
		if got := Execute(context.Background(), nil, producer).Code; got != test.code {
			t.Errorf("Execute error code = %q, want %q", got, test.code)
		}
	}
}

func TestAllRegisteredPayloadsProduceStructuredResults(t *testing.T) {
	for _, entry := range Visible() {
		if _, ok := entry.Payload.(ResultProducer); !ok {
			t.Errorf("payload %q does not implement ResultProducer", entry.Name)
		}
	}
}
