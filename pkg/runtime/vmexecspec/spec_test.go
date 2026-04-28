package vmexecspec

import "testing"

func TestBuildAndParseLinux(t *testing.T) {
	spec := BuildLinux("id && hostname")
	osType, command, ok := Parse(spec)
	if !ok {
		t.Fatalf("Parse(%q) failed", spec)
	}
	if osType != "linux" || command != "id && hostname" {
		t.Fatalf("Parse(%q) = (%q, %q), want (linux, id && hostname)", spec, osType, command)
	}
}

func TestBuildAndParseWindows(t *testing.T) {
	spec := BuildWindows("whoami")
	osType, command, ok := Parse(spec)
	if !ok {
		t.Fatalf("Parse(%q) failed", spec)
	}
	if osType != "windows" || command != "whoami" {
		t.Fatalf("Parse(%q) = (%q, %q), want (windows, whoami)", spec, osType, command)
	}
}
