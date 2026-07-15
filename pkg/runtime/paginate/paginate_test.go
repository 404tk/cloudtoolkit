package paginate

import (
	"context"
	"errors"
	"testing"
)

func TestFetchReturnsContextErrorWithPartialItems(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	items, err := Fetch[int, int](ctx, func(context.Context, int) (Page[int, int], error) {
		calls++
		cancel()
		return Page[int, int]{Items: []int{1}, Next: 1}, nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fetch() error = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Fatalf("Fetch() calls = %d, want 1", calls)
	}
	if len(items) != 1 || items[0] != 1 {
		t.Fatalf("Fetch() items = %#v, want [1]", items)
	}
}
