package replay

import (
	"strconv"
	"strings"
)

// Window is the half-open index range [Start, End) used to paginate fixtures.
type Window struct {
	Start int
	End   int
}

// PageWindow computes the slice for 1-based pagination.
func PageWindow(total, pageNumber, pageSize int) Window {
	if pageNumber <= 0 {
		pageNumber = 1
	}
	if pageSize <= 0 {
		pageSize = total
	}
	start := (pageNumber - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return Window{Start: start, End: end}
}

// OffsetWindow computes the slice for offset/limit-style pagination.
func OffsetWindow(total, offset, size int) Window {
	if offset < 0 {
		offset = 0
	}
	if size <= 0 {
		size = total
	}
	start := offset
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	return Window{Start: start, End: end}
}

// ParseInt parses value or returns fallback when empty / unparsable / non-positive.
func ParseInt(value string, fallback int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

// FirstNonEmpty returns the first trimmed non-empty value.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if v := strings.TrimSpace(value); v != "" {
			return v
		}
	}
	return ""
}

// NonEmptyStrings returns trimmed copies of every non-empty input value.
func NonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if v := strings.TrimSpace(value); v != "" {
			out = append(out, v)
		}
	}
	return out
}
