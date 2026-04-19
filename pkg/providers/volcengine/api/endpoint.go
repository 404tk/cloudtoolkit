package api

import "strings"

const (
	DefaultRegion    = "cn-beijing"
	defaultSiteStack = "volcengineapi"
)

var globalServices = map[string]struct{}{
	"billing": {},
	"iam":     {},
}

// ResolveEndpoint returns the HTTPS base URL for a Volcengine OpenAPI service.
func ResolveEndpoint(service, region, siteStack string) string {
	service = strings.ToLower(strings.TrimSpace(service))
	region = strings.TrimSpace(region)
	if region == "all" {
		region = DefaultRegion
	}
	stack := strings.TrimSpace(siteStack)
	if stack == "" {
		stack = defaultSiteStack
	}

	host := "open." + stack + ".com"
	switch {
	case service == "":
	case isGlobalService(service):
		host = service + "." + stack + ".com"
	case region != "":
		host = service + "." + region + "." + stack + ".com"
	}
	return "https://" + host
}

func isGlobalService(service string) bool {
	_, ok := globalServices[strings.ToLower(strings.TrimSpace(service))]
	return ok
}
