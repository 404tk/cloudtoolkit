package processbar

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

var (
	outputMu sync.RWMutex
	outputW  io.Writer
)

func SetOutput(w io.Writer) {
	outputMu.Lock()
	defer outputMu.Unlock()
	outputW = w
}

func writer() io.Writer {
	outputMu.RLock()
	defer outputMu.RUnlock()
	if outputW == nil {
		return os.Stdout
	}
	return outputW
}

// RegionTracker renders one "[region] N found." line per region iteration,
// collapsing zero-count regions into a single refreshing line and printing a
// new line for regions that actually returned items. Safe for concurrent Update.
type RegionTracker struct {
	mu         sync.Mutex
	prevLength int
	flag       bool
	count      int
}

func NewRegionTracker() *RegionTracker { return &RegionTracker{} }

// Update prints progress for a region and updates internal state.
func (t *RegionTracker) Update(region string, newItems int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prevLength, t.flag = regionPrint(region, newItems, t.prevLength, t.flag)
	t.count += newItems
}

// Count returns the total number of items observed across regions.
func (t *RegionTracker) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.count
}

// Finish clears the in-place progress line if the last Update did not print
// a newline. Call via defer.
func (t *RegionTracker) Finish() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.flag {
		fmt.Fprintf(writer(), "\n\033[F\033[K")
	}
}

// CountTracker renders a single "[tag] N found." line that refreshes in place.
// Used for long-running iterations over a single scope (e.g. object listing in
// one bucket). Safe for concurrent Update.
type CountTracker struct {
	mu         sync.Mutex
	prevLength int
}

func NewCountTracker() *CountTracker { return &CountTracker{} }

// Update refreshes the in-place counter line for tag.
func (t *CountTracker) Update(tag string, count int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	progress := fmt.Sprintf("[%s] %d found.", tag, count)
	progress += padTo(t.prevLength, len(progress))
	fmt.Fprintf(writer(), "\r%s", progress)
	t.prevLength = len(progress)
}

// Finish returns the cursor to column 0 so subsequent output starts cleanly.
// Call via defer.
func (t *CountTracker) Finish() {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Fprint(writer(), "\r")
	t.prevLength = 0
}

func regionPrint(region string, count, prev int, flag bool) (int, bool) {
	progress := fmt.Sprintf("[%s] %d found.", region, count)
	if count == 0 {
		if flag {
			fmt.Fprint(writer(), progress)
		} else {
			progress += padTo(prev, len(progress))
			fmt.Fprintf(writer(), "\r%s", progress)
		}
		flag = false
	} else {
		if flag {
			fmt.Fprintln(writer(), progress)
		} else {
			progress += padTo(prev, len(progress))
			fmt.Fprintf(writer(), "\r%s\n", progress)
		}
		flag = true
	}
	return len(progress), flag
}

func padTo(prev, current int) string {
	if prev > current {
		return strings.Repeat(" ", prev-current)
	}
	return ""
}
