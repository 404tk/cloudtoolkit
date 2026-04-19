package api

import "testing"

func TestResolveHost(t *testing.T) {
	if got := ResolveHost("iam"); got != "iam.jdcloud-api.com" {
		t.Fatalf("unexpected host: %s", got)
	}
	if got := ResolveHost(" "); got != "" {
		t.Fatalf("unexpected empty-service host: %s", got)
	}
}

func TestResolveSigningRegion(t *testing.T) {
	if got := ResolveSigningRegion(""); got != DefaultSigningRegion {
		t.Fatalf("unexpected default region: %s", got)
	}
	if got := ResolveSigningRegion("cn-north-1"); got != "cn-north-1" {
		t.Fatalf("unexpected resolved region: %s", got)
	}
}
