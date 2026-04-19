package api

import (
	"regexp"
	"testing"
)

func TestNewUUIDv4(t *testing.T) {
	got, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4() error = %v", err)
	}
	const pattern = "^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
	if !regexp.MustCompile(pattern).MatchString(got) {
		t.Fatalf("unexpected uuid format: %s", got)
	}
}
