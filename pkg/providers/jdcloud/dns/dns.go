// Package dns wraps JDCloud domainservice DescribeDomains +
// DescribeResourceRecord for the cloudlist `domain` asset.
//
// Each schema.Domain row carries one JDCloud DNS zone with its records
// flattened underneath. Only A / AAAA / CNAME / MX / NS / TXT records are
// surfaced — the same record-type shortlist used by the AWS / GCP / Azure
// DNS drivers.
package dns

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	defaultRegion = "cn-north-1"
	pageSize      = 100
	maxPages      = 50
)

type Driver struct {
	Client *api.Client
	Region string
}

func (d *Driver) requestRegion() string {
	if r := strings.TrimSpace(d.Region); r != "" && r != "all" {
		return r
	}
	return defaultRegion
}

// GetDomains lists JDCloud DNS domains and their records.
func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	out := []schema.Domain{}
	if d == nil || d.Client == nil {
		return out, errors.New("jdcloud dns: nil api client")
	}
	logger.Info("List JDCloud DNS domains ...")
	region := d.requestRegion()
	for page := 1; page <= maxPages; page++ {
		resp, err := d.Client.DescribeDomains(ctx, region, page, pageSize)
		if err != nil {
			return out, err
		}
		for _, info := range resp.Result.DataList {
			records, err := d.listRecords(ctx, region, info.ID)
			if err != nil {
				return out, err
			}
			out = append(out, schema.Domain{
				DomainName: info.DomainName,
				Records:    records,
			})
		}
		if len(resp.Result.DataList) < pageSize {
			break
		}
	}
	return out, nil
}

func (d *Driver) listRecords(ctx context.Context, region string, domainID int) ([]schema.Record, error) {
	out := []schema.Record{}
	if domainID == 0 {
		return out, nil
	}
	id := strconv.Itoa(domainID)
	for page := 1; page <= maxPages; page++ {
		resp, err := d.Client.DescribeResourceRecord(ctx, region, id, page, pageSize)
		if err != nil {
			return out, err
		}
		for _, rr := range resp.Result.DataList {
			if !recordTypeAllowed(rr.Type) {
				continue
			}
			out = append(out, schema.Record{
				RR:     rr.HostRecord,
				Type:   rr.Type,
				Value:  rr.HostValue,
				Status: rrStatusLabel(rr.ResolvingStatus),
			})
		}
		if len(resp.Result.DataList) < pageSize {
			break
		}
	}
	return out, nil
}

// recordTypeAllowed mirrors aws/gcp dns: only surface record types CSPM
// detectors usually inspect.
func recordTypeAllowed(t string) bool {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "A", "AAAA", "CNAME", "MX", "NS", "TXT":
		return true
	}
	return false
}

// rrStatusLabel translates JDCloud RRInfo.resolvingStatus enum codes into the
// human readable labels surfaced by `cloudlist`. JDCloud documents only "2"
// (normal) and "4" (paused) for record-level status; unknown codes pass
// through verbatim so future values stay visible to operators.
func rrStatusLabel(code string) string {
	switch strings.TrimSpace(code) {
	case "":
		return ""
	case "2":
		return "Enable"
	case "4":
		return "Pause"
	default:
		return code
	}
}
