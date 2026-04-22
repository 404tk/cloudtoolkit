package api

import "strings"

const (
	DefaultRegion    = "cn-beijing"
	defaultSiteStack = "volcengineapi"
)

var globalServices = map[string]struct{}{
	"billing": {},
	"dns":     {},
	"iam":     {},
}

var regionalServiceHostAliases = map[string]string{
	"rds_mysql":      "rds-mysql",
	"rds_postgresql": "rds-postgresql",
	"rds_mssql":      "rds-mssql",
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
		host = endpointHostPrefix(service) + "." + region + "." + stack + ".com"
	}
	return "https://" + host
}

func endpointHostPrefix(service string) string {
	if alias, ok := regionalServiceHostAliases[service]; ok {
		return alias
	}
	return service
}

func isGlobalService(service string) bool {
	_, ok := globalServices[strings.ToLower(strings.TrimSpace(service))]
	return ok
}
