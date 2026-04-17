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
//   https://{service}.{region}.myhuaweicloud.com
//
// with a small number of services (bss, iam) historically served from global
// or partition-specific hosts. Those exceptions are encoded here.
package endpoint

import "fmt"

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
	case "obs":
		return fmt.Sprintf("https://obs.%s.myhuaweicloud.com", region)
	}
	return fmt.Sprintf("https://%s.%s.myhuaweicloud.com", service, region)
}
