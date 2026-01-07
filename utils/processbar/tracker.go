package processbar

import "fmt"

// RegionTracker encapsulates region iteration progress tracking state.
type RegionTracker struct {
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
	t.prevLength, t.flag = RegionPrint(region, newItems, t.prevLength, t.flag)
	t.count += newItems
}

// Count returns the total count of items found across all regions.
func (t *RegionTracker) Count() int {
	return t.count
}

// Finish cleans up the progress line if needed.
// Should be called with defer after creating the tracker.
func (t *RegionTracker) Finish() {
	if !t.flag {
		fmt.Printf("\n\033[F\033[K")
	}
}
