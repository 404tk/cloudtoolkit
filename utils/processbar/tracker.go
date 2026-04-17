package processbar

import (
	"fmt"
	"sync"
)

// RegionTracker encapsulates region iteration progress tracking state.
// Safe for concurrent Update calls.
type RegionTracker struct {
	mu         sync.Mutex
	prevLength int
	flag       bool
	count      int
}

// NewRegionTracker creates a new RegionTracker instance.
func NewRegionTracker() *RegionTracker {
	return &RegionTracker{}
}

// Update prints progress for a region and updates internal state.
// newItems is the count of items found in this region.
func (t *RegionTracker) Update(region string, newItems int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prevLength, t.flag = RegionPrint(region, newItems, t.prevLength, t.flag)
	t.count += newItems
}

// Count returns the total count of items found across all regions.
func (t *RegionTracker) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.count
}

// Finish cleans up the progress line if needed.
// Should be called with defer after creating the tracker.
func (t *RegionTracker) Finish() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.flag {
		fmt.Printf("\n\033[F\033[K")
	}
}
