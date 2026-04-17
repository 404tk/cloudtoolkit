// Package paginate provides a thin generic skeleton for SDK pagination loops.
//
// It removes the "for { call API; append; advance cursor; break-on-last }"
// boilerplate that every provider's region enumerator re-implements. The
// caller owns everything SDK-specific (cursor type, end-of-stream detection,
// item mapping); this package just drives the loop.
package paginate

import "context"

// Page is one API response's worth of items plus the cursor to fetch the next.
// Set Done=true on the final page — Next is ignored when Done is true.
type Page[Item, Cursor any] struct {
	Items []Item
	Next  Cursor
	Done  bool
}

// Fetch calls fn repeatedly starting from the zero value of Cursor, appending
// each page's Items to the result. It stops when fn returns Done=true, returns
// an error, or ctx is cancelled (already-collected items are returned, err nil
// on cancellation — matches the partial-failure convention used elsewhere).
func Fetch[Item, Cursor any](
	ctx context.Context,
	fn func(ctx context.Context, cursor Cursor) (Page[Item, Cursor], error),
) ([]Item, error) {
	var (
		cursor Cursor
		out    []Item
	)
	for {
		select {
		case <-ctx.Done():
			return out, nil
		default:
		}
		page, err := fn(ctx, cursor)
		if err != nil {
			return out, err
		}
		out = append(out, page.Items...)
		if page.Done {
			return out, nil
		}
		cursor = page.Next
	}
}
