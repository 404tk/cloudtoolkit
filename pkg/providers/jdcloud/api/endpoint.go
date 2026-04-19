package api

import "strings"

const DefaultSigningRegion = "jdcloud-api"

// ResolveHost returns the service endpoint host used by J1 actions.
func ResolveHost(service string) string {
	service = strings.ToLower(strings.TrimSpace(service))
	if service == "" {
		return ""
	}
	return service + ".jdcloud-api.com"
}

// ResolveSigningRegion returns the region that participates in signature
// derivation. Global IAM calls fall back to the literal "jdcloud-api".
func ResolveSigningRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		return DefaultSigningRegion
	}
	return region
}
