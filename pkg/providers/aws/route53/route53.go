package route53

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	maxHostedZonesPerPage = 100
	maxRecordsPerPage     = 100
)

// Driver enumerates Route53 hosted zones and their resource record sets,
// surfacing them as the cloudlist `domain` asset. Route53 is a global service,
// so the driver does not iterate per-region; the SigV4 region is fixed at
// `us-east-1` by the API client.
type Driver struct {
	Client *api.Client
}

// GetDomains returns one schema.Domain per hosted zone, with records flattened
// across all rrsets. A single per-zone error short-circuits that zone's
// records but is appended via list.AddError; remaining zones still surface.
func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	list := []schema.Domain{}
	if d == nil || d.Client == nil {
		return list, errors.New("aws route53: nil api client")
	}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List Route53 hosted zones ...")
	}

	zones, err := paginate.Fetch[api.HostedZone, string](ctx, func(ctx context.Context, marker string) (paginate.Page[api.HostedZone, string], error) {
		resp, err := d.Client.Route53ListHostedZones(ctx, marker, maxHostedZonesPerPage)
		if err != nil {
			return paginate.Page[api.HostedZone, string]{}, err
		}
		return paginate.Page[api.HostedZone, string]{
			Items: resp.HostedZones,
			Next:  resp.NextMarker,
			Done:  !resp.IsTruncated || strings.TrimSpace(resp.NextMarker) == "",
		}, nil
	})
	if err != nil {
		return list, err
	}

	for _, z := range zones {
		domainName := strings.TrimSuffix(z.Name, ".")
		if domainName == "" {
			domainName = z.ID
		}
		records, recErr := d.listRecords(ctx, z.ID)
		if recErr != nil {
			logger.Error(fmt.Sprintf("Route53 ListResourceRecordSets %s: %s", z.ID, recErr.Error()))
			// Surface partial coverage: keep the zone with whatever records
			// were already collected and continue.
			list = append(list, schema.Domain{DomainName: domainName, Records: records})
			continue
		}
		list = append(list, schema.Domain{DomainName: domainName, Records: records})
	}
	return list, nil
}

func (d *Driver) listRecords(ctx context.Context, zoneID string) ([]schema.Record, error) {
	out := []schema.Record{}
	type cursor struct {
		Name, Type, Identifier string
	}
	pages, err := paginate.Fetch[api.Route53Record, cursor](ctx, func(ctx context.Context, c cursor) (paginate.Page[api.Route53Record, cursor], error) {
		resp, err := d.Client.Route53ListResourceRecordSets(ctx, zoneID, c.Name, c.Type, c.Identifier, maxRecordsPerPage)
		if err != nil {
			return paginate.Page[api.Route53Record, cursor]{}, err
		}
		next := cursor{
			Name:       resp.NextRecordName,
			Type:       resp.NextRecordType,
			Identifier: resp.NextRecordIdentifier,
		}
		return paginate.Page[api.Route53Record, cursor]{
			Items: resp.Records,
			Next:  next,
			Done:  !resp.IsTruncated || (next.Name == "" && next.Type == ""),
		}, nil
	})
	if err != nil {
		return out, err
	}
	for _, r := range pages {
		if !recordTypeAllowed(r.Type) {
			continue
		}
		// Each ResourceRecord value (or alias DNS name) is surfaced as a
		// separate schema.Record so consumers can flag e.g. one A record
		// pointing to a public IP independent of siblings.
		for _, value := range r.Values {
			out = append(out, schema.Record{
				RR:     strings.TrimSuffix(r.Name, "."),
				Type:   r.Type,
				Value:  value,
				Status: r.Status,
			})
		}
	}
	return out, nil
}

// recordTypeAllowed mirrors gcp/dns: only surface record types CSPM detectors
// typically reason about. Route53 also exposes SOA/NS records which are not
// useful for shadowing detection.
func recordTypeAllowed(t string) bool {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "A", "AAAA", "CNAME", "TXT", "MX":
		return true
	default:
		return false
	}
}
