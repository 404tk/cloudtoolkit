package dns

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

// defaultDNSRegion is the public-zone control plane Huawei surfaces zones
// from. The DNS service is per-region-routed but public zones are
// account-scoped, so we issue a single call against the default control
// region. Operators with private zones in other regions can still discover
// the zone IDs via the per-region recordsets URL if they extend Regions.
const defaultDNSRegion = "cn-north-4"

// Driver enumerates Huawei Cloud DNS public zones and their record sets,
// returning them as the cloudlist `domain` asset.
//
// Private DNS zones are out of scope for now: their endpoint differs and the
// CSPM signal value (DNS shadowing, dangling CNAMEs) is concentrated on the
// internet-facing public zones.
type Driver struct {
	Cred    auth.Credential
	Regions []string
	Client  *api.Client
}

func (d *Driver) client() *api.Client {
	if d.Client == nil {
		d.Client = api.NewClient(d.Cred)
	}
	return d.Client
}

// GetDomains returns one schema.Domain per zone, with records flattened over
// all record sets. A failure on a single zone's record-set listing is
// recorded but does not abort the whole walk; a top-level zone-listing
// failure returns an error so the caller can surface it via list.AddError.
func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	list := []schema.Domain{}
	if d == nil {
		return list, errors.New("huawei dns: nil driver")
	}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List Huawei DNS zones ...")
	}

	region := d.region()
	zones, err := d.listZones(ctx, region)
	if err != nil {
		return list, err
	}
	for _, z := range zones {
		zoneName := strings.TrimSuffix(strings.TrimSpace(z.Name), ".")
		if zoneName == "" {
			continue
		}
		records, recErr := d.listRecordSets(ctx, region, z.ID, zoneName)
		if recErr != nil {
			logger.Error(fmt.Sprintf("Huawei DNS recordsets %s: %s", z.ID, recErr.Error()))
			list = append(list, schema.Domain{DomainName: zoneName, Records: records})
			continue
		}
		list = append(list, schema.Domain{DomainName: zoneName, Records: records})
	}
	return list, nil
}

func (d *Driver) listZones(ctx context.Context, region string) ([]api.DNSZone, error) {
	var resp api.ListZonesResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "dns",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v2/zones",
		Idempotent: true,
	}, &resp); err != nil {
		return nil, err
	}
	return resp.Zones, nil
}

func (d *Driver) listRecordSets(ctx context.Context, region, zoneID, zoneName string) ([]schema.Record, error) {
	out := []schema.Record{}
	if zoneID = strings.TrimSpace(zoneID); zoneID == "" {
		return out, errors.New("huawei dns: empty zone id")
	}
	var resp api.ListRecordSetsResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "dns",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v2/zones/" + zoneID + "/recordsets",
		Idempotent: true,
	}, &resp); err != nil {
		return out, err
	}
	for _, rs := range resp.RecordSets {
		typ := strings.ToUpper(strings.TrimSpace(rs.Type))
		if !recordTypeAllowed(typ) {
			continue
		}
		// Huawei encodes recordset names with a trailing dot to mirror DNS
		// canonical form. Strip it so cloudlist output stays consistent with
		// the trimmed zoneName above.
		rr := strings.TrimSuffix(strings.TrimSpace(rs.Name), ".")
		for _, value := range rs.Records {
			out = append(out, schema.Record{
				RR:     rr,
				Type:   typ,
				Value:  strings.TrimSpace(value),
				Status: strings.TrimSpace(rs.Status),
			})
		}
	}
	return out, nil
}

func (d *Driver) region() string {
	for _, r := range d.Regions {
		if r = strings.TrimSpace(r); r != "" && r != "all" {
			return r
		}
	}
	if r := strings.TrimSpace(d.Cred.Region); r != "" && r != "all" {
		return r
	}
	return defaultDNSRegion
}

func recordTypeAllowed(t string) bool {
	switch t {
	case "A", "AAAA", "CNAME", "TXT", "MX":
		return true
	default:
		return false
	}
}
