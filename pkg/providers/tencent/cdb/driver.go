package cdb

import (
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

type Driver struct {
	Credential    auth.Credential
	Region        string
	clientOptions []api.Option
	partialErr    error
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Credential, d.clientOptions...)
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) PartialError() error {
	return d.partialErr
}

func addRegion(regions *[]string, region string) {
	if region == "" {
		return
	}
	for _, existing := range *regions {
		if existing == region {
			return
		}
	}
	*regions = append(*regions, region)
}

func normalizedRegion(region string) string {
	switch region {
	case "", "all":
		return api.DefaultRegion
	default:
		return region
	}
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func formatAddressInt64(host *string, port *int64) string {
	if host == nil || port == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", *host, *port)
}

func formatAddressUint64(host *string, port *uint64) string {
	if host == nil || port == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", *host, *port)
}

func unsupportedRegion(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		code := strings.TrimSpace(apiErr.Code)
		return code == "UnsupportedRegion" || strings.HasSuffix(code, ".UnsupportedRegion")
	}
	return false
}

func mergeRegionErrors(base, extra map[string]error) map[string]error {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	merged := make(map[string]error, len(base)+len(extra))
	for region, err := range base {
		if err != nil {
			merged[region] = err
		}
	}
	for region, err := range extra {
		if err != nil {
			merged[region] = err
		}
	}
	return merged
}
