package dns

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

// Driver enumerates Azure public DNS zones (`Microsoft.Network/dnsZones`)
// and their record sets, mapping them to the cloudlist `domain` asset.
//
// Private DNS zones live under `Microsoft.Network/privateDnsZones` with a
// separate API version; they are out of scope here because the highest CSPM
// signal value (DNS shadowing, dangling CNAMEs) lives on internet-facing
// public zones.
type Driver struct {
	Client          *azapi.Client
	SubscriptionIDs []string
}

// GetDomains returns one schema.Domain per public DNS zone. A failure on any
// single zone's record-set listing is logged and the zone is surfaced with
// whatever records were collected so far; subscription-level failures are
// returned to the caller and aggregated via list.AddError upstream.
func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	list := []schema.Domain{}
	if d == nil || d.Client == nil {
		return list, fmt.Errorf("azure dns: nil api client")
	}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List Azure DNS zones ...")
	}

	for _, subscription := range d.SubscriptionIDs {
		zones, err := d.listZones(ctx, subscription)
		if err != nil {
			logger.Error(fmt.Sprintf("List Azure DNS zones in subscription %s failed: %s", subscription, err.Error()))
			return list, err
		}
		for _, z := range zones {
			zoneName := strings.TrimSpace(z.Name)
			if zoneName == "" {
				continue
			}
			records, recErr := d.listRecordSets(ctx, z.ID)
			if recErr != nil {
				logger.Error(fmt.Sprintf("Azure DNS recordSets %s: %s", z.ID, recErr.Error()))
				list = append(list, schema.Domain{DomainName: zoneName, Records: records})
				continue
			}
			list = append(list, schema.Domain{DomainName: zoneName, Records: records})
		}
	}
	return list, nil
}

func (d *Driver) listZones(ctx context.Context, subscription string) ([]azapi.DNSZone, error) {
	pager := azapi.NewPager[azapi.DNSZone](d.Client, azapi.Request{
		Method:     http.MethodGet,
		Path:       fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Network/dnsZones", subscription),
		Query:      url.Values{"api-version": {azapi.DNSAPIVersion}},
		Idempotent: true,
	})
	return pager.All(ctx)
}

func (d *Driver) listRecordSets(ctx context.Context, zoneID string) ([]schema.Record, error) {
	out := []schema.Record{}
	zoneID = strings.TrimSpace(zoneID)
	if zoneID == "" {
		return out, fmt.Errorf("azure dns: empty zone id")
	}
	// ARM list-by-zone path is `<zoneID>/recordsets` (lowercased "recordsets"
	// per the 2018-05-01 spec).
	pager := azapi.NewPager[azapi.DNSRecordSet](d.Client, azapi.Request{
		Method:     http.MethodGet,
		Path:       strings.TrimRight(zoneID, "/") + "/recordsets",
		Query:      url.Values{"api-version": {azapi.DNSAPIVersion}},
		Idempotent: true,
	})
	rrsets, err := pager.All(ctx)
	if err != nil {
		return out, err
	}
	for _, rs := range rrsets {
		typ := recordTypeOf(rs.Type)
		if !recordTypeAllowed(typ) {
			continue
		}
		rr := strings.TrimSpace(rs.Name)
		// Azure record-set names are zone-relative (e.g. "@", "www"). Materialize
		// the FQDN when the property is provided so downstream consumers don't
		// need zone context.
		if fqdn := strings.TrimSuffix(strings.TrimSpace(rs.Properties.FQDN), "."); fqdn != "" {
			rr = fqdn
		}
		for _, value := range valuesOf(typ, rs.Properties) {
			out = append(out, schema.Record{
				RR:     rr,
				Type:   typ,
				Value:  value,
				Status: "ENABLE",
			})
		}
	}
	return out, nil
}

// recordTypeOf trims the Microsoft.Network/dnszones/* type prefix returned by
// ARM, leaving the bare record type ("A", "CNAME", ...).
func recordTypeOf(t string) string {
	t = strings.TrimSpace(t)
	if idx := strings.LastIndex(t, "/"); idx >= 0 && idx+1 < len(t) {
		return strings.ToUpper(t[idx+1:])
	}
	return strings.ToUpper(t)
}

func recordTypeAllowed(t string) bool {
	switch t {
	case "A", "AAAA", "CNAME", "TXT", "MX":
		return true
	default:
		return false
	}
}

func valuesOf(typ string, p azapi.DNSRecordSetProps) []string {
	switch typ {
	case "A":
		out := make([]string, 0, len(p.ARecords))
		for _, r := range p.ARecords {
			if v := strings.TrimSpace(r.IPv4Address); v != "" {
				out = append(out, v)
			}
		}
		return out
	case "AAAA":
		out := make([]string, 0, len(p.AAAARecords))
		for _, r := range p.AAAARecords {
			if v := strings.TrimSpace(r.IPv6Address); v != "" {
				out = append(out, v)
			}
		}
		return out
	case "CNAME":
		if p.CNAMERecord == nil {
			return nil
		}
		v := strings.TrimSpace(p.CNAMERecord.CNAME)
		if v == "" {
			return nil
		}
		return []string{strings.TrimSuffix(v, ".")}
	case "MX":
		out := make([]string, 0, len(p.MXRecords))
		for _, r := range p.MXRecords {
			if r.Exchange == "" {
				continue
			}
			out = append(out, fmt.Sprintf("%d %s", r.Preference, strings.TrimSuffix(r.Exchange, ".")))
		}
		return out
	case "TXT":
		out := []string{}
		for _, r := range p.TXTRecords {
			if len(r.Value) == 0 {
				continue
			}
			out = append(out, strings.Join(r.Value, ""))
		}
		return out
	}
	return nil
}
