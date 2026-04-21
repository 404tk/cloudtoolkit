// Package regionrun runs per-region enumeration callbacks in parallel with a
// bounded concurrency. It matches the existing "partial failure" semantics of
// the rest of the codebase: a single region's error does not cancel siblings,
// and errors are returned keyed by region so the caller can surface them via
// Resources.AddError.
package regionrun

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

// DefaultConcurrency is the fan-out used when the caller passes concurrency <= 0.
const DefaultConcurrency = 6

type PartialError struct {
	errs map[string]error
}

func Wrap(errs map[string]error) error {
	if len(errs) == 0 {
		return nil
	}
	copied := make(map[string]error, len(errs))
	for region, err := range errs {
		if err == nil {
			continue
		}
		copied[region] = err
	}
	if len(copied) == 0 {
		return nil
	}
	return PartialError{errs: copied}
}

func (e PartialError) Error() string {
	regions := e.sortedRegions()
	parts := make([]string, 0, len(regions))
	for _, region := range regions {
		if err := e.errs[region]; err != nil {
			parts = append(parts, fmt.Sprintf("%s: %v", region, err))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("partial region errors: %s", strings.Join(parts, "; "))
}

func (e PartialError) ResourceErrors(scope string) []schema.ResourceError {
	regions := e.sortedRegions()
	items := make([]schema.ResourceError, 0, len(regions))
	for _, region := range regions {
		err := e.errs[region]
		if err == nil {
			continue
		}
		itemScope := region
		if strings.TrimSpace(scope) != "" {
			itemScope = fmt.Sprintf("%s/%s", scope, region)
		}
		items = append(items, schema.ResourceError{
			Scope:   itemScope,
			Message: err.Error(),
		})
	}
	return items
}

func (e PartialError) sortedRegions() []string {
	regions := make([]string, 0, len(e.errs))
	for region := range e.errs {
		regions = append(regions, region)
	}
	sort.Strings(regions)
	return regions
}

// ForEach invokes fn for each region, capped at `concurrency` in-flight calls.
// If tracker is non-nil, each region's completion emits a progress update
// (serialised — tracker updates are not thread-safe on their own caller side).
// Returns the aggregated slice and a map of region -> error for regions that
// failed. Honours ctx.Done() by stopping dispatch of new regions; in-flight
// fns receive the same ctx and are expected to bail out.
func ForEach[T any](
	ctx context.Context,
	regions []string,
	concurrency int,
	tracker *processbar.RegionTracker,
	fn func(ctx context.Context, region string) ([]T, error),
) ([]T, map[string]error) {
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}
	sem := make(chan struct{}, concurrency)
	var (
		mu   sync.Mutex
		out  []T
		errs = map[string]error{}
		wg   sync.WaitGroup
	)
	for _, r := range regions {
		select {
		case <-ctx.Done():
			wg.Wait()
			return out, errs
		case sem <- struct{}{}:
		}
		wg.Add(1)
		region := r
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			items, err := fn(ctx, region)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs[region] = err
			}
			before := len(out)
			out = append(out, items...)
			if tracker != nil {
				tracker.Update(region, len(out)-before)
			}
		}()
	}
	wg.Wait()
	return out, errs
}
