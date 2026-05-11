// Package endpoint builds Huawei Cloud service endpoint URLs without going
// through the official SDK's region.ValueOf() lookup.
//
// Every service SDK under huaweicloud-sdk-go-v3/services/<svc>/<ver>/region
// ships a hardcoded region list. Passing a region the list does not know
// panics. Several of those lists are incomplete or stale (e.g. missing newer
// availability zones). We sidestep the panic entirely by using the SDK's
// WithEndpoint(url) builder option with a URL we construct ourselves.
//
// Huawei Cloud endpoints follow a uniform pattern for service+region
// combinations:
//
//	https://{service}.{region}.myhuaweicloud.com
//
// with a small number of global, partition-specific, or regional endpoint
// exceptions. Those exceptions are encoded here.
package endpoint

import "fmt"

type endpointKey struct {
	service string
	region  string
}

var regionalEndpointOverrides = map[endpointKey]string{
	{service: "cts", region: "eu-west-101"}: "https://cts.eu-west-101.myhuaweicloud.eu",
	{service: "iam", region: "eu-west-101"}: "https://iam.eu-west-101.myhuaweicloud.eu",
	{service: "lts", region: "eu-west-101"}: "https://lts.eu-west-101.myhuaweicloud.eu",
	{service: "rds", region: "eu-west-101"}: "https://rds.eu-west-101.myhuaweicloud.eu",
}

// For returns the endpoint URL to pass to WithEndpoint() for the given
// (service, region) pair. intl toggles between mainland China and
// international billing partitions for the handful of services where they
// differ.
func For(service, region string, intl bool) string {
	switch service {
	case "bss":
		if intl {
			return "https://bss-intl.myhuaweicloud.com"
		}
		return "https://bss.myhuaweicloud.com"
	case "coc":
		switch region {
		case "eu-west-101":
			return "https://coc-eu-west-101-open-api.myhuaweicloud.eu"
		case "ap-southeast-3":
			return "https://coc-intl.myhuaweicloud.com"
		default:
			if intl {
				return "https://coc-intl.myhuaweicloud.com"
			}
			return "https://coc.myhuaweicloud.com"
		}
	case "obs":
		return fmt.Sprintf("https://obs.%s.myhuaweicloud.com", region)
	}
	if endpoint, ok := regionalEndpointOverrides[endpointKey{service: service, region: region}]; ok {
		return endpoint
	}
	return fmt.Sprintf("https://%s.%s.myhuaweicloud.com", service, region)
}
