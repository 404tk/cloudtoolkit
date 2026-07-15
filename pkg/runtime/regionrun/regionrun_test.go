package regionrun

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestForEachCancelledContextStopsDispatchAndRecordsErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var calls atomic.Int32
	items, errs := ForEach(ctx, []string{"r1", "r2", "r3"}, 1, nil, func(context.Context, string) ([]int, error) {
		calls.Add(1)
		return []int{1}, nil
	})

	if calls.Load() != 0 {
		t.Fatalf("ForEach() dispatched %d callbacks after cancellation", calls.Load())
	}
	if len(items) != 0 {
		t.Fatalf("ForEach() items = %#v, want none", items)
	}
	for _, region := range []string{"r1", "r2", "r3"} {
		if !errors.Is(errs[region], context.Canceled) {
			t.Fatalf("ForEach() error[%s] = %v, want context.Canceled", region, errs[region])
		}
	}
}
