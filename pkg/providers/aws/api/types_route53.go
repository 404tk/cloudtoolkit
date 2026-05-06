package api

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
)

// AWS Route53 is a global REST/XML service. The data plane uses path-based
// resources under `/2013-04-01/hostedzone[/{Id}/rrset]` rather than the
// query-string action style that Query-protocol services (IAM/STS) use.
const route53APIVersion = "2013-04-01"

// HostedZone is the public/private DNS zone record returned by ListHostedZones.
type HostedZone struct {
	ID              string
	Name            string
	PrivateZone     bool
	ResourceCount   int64
	CallerReference string
	Comment         string
}

// ListHostedZonesOutput is the typed result of Route53 ListHostedZones.
type ListHostedZonesOutput struct {
	HostedZones []HostedZone
	NextMarker  string
	IsTruncated bool
	RequestID   string
}

// Route53Record is a single resource-record-set entry under a hosted zone.
type Route53Record struct {
	Name string
	Type string
	TTL  int64
	// Values aggregates ResourceRecords[].Value entries plus any AliasTarget DNS
	// name when the record is an alias.
	Values  []string
	Status  string
	IsAlias bool
}

// ListResourceRecordSetsOutput is the typed result of Route53
// ListResourceRecordSets.
type ListResourceRecordSetsOutput struct {
	Records              []Route53Record
	NextRecordName       string
	NextRecordType       string
	NextRecordIdentifier string
	IsTruncated          bool
	RequestID            string
}

type hostedZoneWire struct {
	ID     string                `xml:"Id"`
	Name   string                `xml:"Name"`
	Config hostedZoneConfigWire  `xml:"Config"`
	RRSC   int64                 `xml:"ResourceRecordSetCount"`
	CRef   string                `xml:"CallerReference"`
}

type hostedZoneConfigWire struct {
	Comment     string `xml:"Comment"`
	PrivateZone bool   `xml:"PrivateZone"`
}

type listHostedZonesResponse struct {
	XMLName     xml.Name         `xml:"ListHostedZonesResponse"`
	HostedZones []hostedZoneWire `xml:"HostedZones>HostedZone"`
	Marker      string           `xml:"Marker"`
	IsTruncated bool             `xml:"IsTruncated"`
	NextMarker  string           `xml:"NextMarker"`
	MaxItems    string           `xml:"MaxItems"`
}

type rrsetResourceRecordWire struct {
	Value string `xml:"Value"`
}

type rrsetAliasTargetWire struct {
	HostedZoneID         string `xml:"HostedZoneId"`
	DNSName              string `xml:"DNSName"`
	EvaluateTargetHealth bool   `xml:"EvaluateTargetHealth"`
}

type rrsetWire struct {
	Name            string                    `xml:"Name"`
	Type            string                    `xml:"Type"`
	TTL             int64                     `xml:"TTL"`
	SetIdentifier   string                    `xml:"SetIdentifier"`
	ResourceRecords []rrsetResourceRecordWire `xml:"ResourceRecords>ResourceRecord"`
	AliasTarget     *rrsetAliasTargetWire     `xml:"AliasTarget"`
}

type listRecordSetsResponse struct {
	XMLName              xml.Name    `xml:"ListResourceRecordSetsResponse"`
	ResourceRecordSets   []rrsetWire `xml:"ResourceRecordSets>ResourceRecordSet"`
	IsTruncated          bool        `xml:"IsTruncated"`
	MaxItems             string      `xml:"MaxItems"`
	NextRecordName       string      `xml:"NextRecordName"`
	NextRecordType       string      `xml:"NextRecordType"`
	NextRecordIdentifier string      `xml:"NextRecordIdentifier"`
}

// Route53ListHostedZones lists hosted zones in the caller's account. AWS
// Route53 is a global service; the SigV4 region is fixed to `us-east-1` by
// the signer (see normalizeServiceRegion).
func (c *Client) Route53ListHostedZones(ctx context.Context, marker string, maxItems int) (ListHostedZonesOutput, error) {
	query := url.Values{}
	if marker = strings.TrimSpace(marker); marker != "" {
		query.Set("marker", marker)
	}
	if maxItems > 0 {
		query.Set("maxitems", intToString(maxItems))
	}
	var wire listHostedZonesResponse
	err := c.DoRESTXML(ctx, Request{
		Service:    "route53",
		Region:     "us-east-1",
		Method:     http.MethodGet,
		Path:       "/" + route53APIVersion + "/hostedzone",
		Query:      query,
		Idempotent: true,
	}, &wire)
	if err != nil {
		return ListHostedZonesOutput{}, err
	}
	out := ListHostedZonesOutput{
		HostedZones: make([]HostedZone, 0, len(wire.HostedZones)),
		IsTruncated: wire.IsTruncated,
		NextMarker:  strings.TrimSpace(wire.NextMarker),
	}
	for _, z := range wire.HostedZones {
		out.HostedZones = append(out.HostedZones, HostedZone{
			ID:              normalizeHostedZoneID(z.ID),
			Name:            strings.TrimSpace(z.Name),
			PrivateZone:     z.Config.PrivateZone,
			ResourceCount:   z.RRSC,
			CallerReference: z.CRef,
			Comment:         z.Config.Comment,
		})
	}
	return out, nil
}

// Route53ListResourceRecordSets lists record sets within a hosted zone.
// `zoneID` may be either the bare ID ("Z2ABCDE") or the prefixed form
// ("/hostedzone/Z2ABCDE") — both are accepted.
func (c *Client) Route53ListResourceRecordSets(ctx context.Context, zoneID, startName, startType, startIdentifier string, maxItems int) (ListResourceRecordSetsOutput, error) {
	id := normalizeHostedZoneID(zoneID)
	if id == "" {
		return ListResourceRecordSetsOutput{}, errEmptyHostedZoneID
	}
	query := url.Values{}
	if startName = strings.TrimSpace(startName); startName != "" {
		query.Set("name", startName)
	}
	if startType = strings.TrimSpace(startType); startType != "" {
		query.Set("type", startType)
	}
	if startIdentifier = strings.TrimSpace(startIdentifier); startIdentifier != "" {
		query.Set("identifier", startIdentifier)
	}
	if maxItems > 0 {
		query.Set("maxitems", intToString(maxItems))
	}
	var wire listRecordSetsResponse
	err := c.DoRESTXML(ctx, Request{
		Service:    "route53",
		Region:     "us-east-1",
		Method:     http.MethodGet,
		Path:       "/" + route53APIVersion + "/hostedzone/" + id + "/rrset",
		Query:      query,
		Idempotent: true,
	}, &wire)
	if err != nil {
		return ListResourceRecordSetsOutput{}, err
	}
	out := ListResourceRecordSetsOutput{
		Records:              make([]Route53Record, 0, len(wire.ResourceRecordSets)),
		IsTruncated:          wire.IsTruncated,
		NextRecordName:       strings.TrimSpace(wire.NextRecordName),
		NextRecordType:       strings.TrimSpace(wire.NextRecordType),
		NextRecordIdentifier: strings.TrimSpace(wire.NextRecordIdentifier),
	}
	for _, r := range wire.ResourceRecordSets {
		rec := Route53Record{
			Name:   strings.TrimSpace(r.Name),
			Type:   strings.TrimSpace(r.Type),
			TTL:    r.TTL,
			Status: "ENABLE",
		}
		for _, v := range r.ResourceRecords {
			if val := strings.TrimSpace(v.Value); val != "" {
				rec.Values = append(rec.Values, val)
			}
		}
		if r.AliasTarget != nil {
			rec.IsAlias = true
			if dns := strings.TrimSpace(r.AliasTarget.DNSName); dns != "" {
				rec.Values = append(rec.Values, "ALIAS "+dns)
			}
		}
		out.Records = append(out.Records, rec)
	}
	return out, nil
}

// normalizeHostedZoneID strips the `/hostedzone/` prefix that Route53 attaches
// to identifiers in API responses, so callers can pass either form into
// downstream calls.
func normalizeHostedZoneID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "/hostedzone/")
	id = strings.TrimPrefix(id, "hostedzone/")
	return strings.TrimSpace(id)
}

func intToString(n int) string {
	// minimal positive-int formatter avoiding strconv import here so the file
	// compiles standalone with the existing import set.
	if n <= 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// errEmptyHostedZoneID is reused so callers get a consistent message rather
// than having Route53 reject the request later.
var errEmptyHostedZoneID = newRoute53Error("hosted zone id required")

// route53Error is a tiny error wrapper kept local so we don't drag a third
// package into types_route53.go just for a sentinel.
type route53Error struct{ msg string }

func (e route53Error) Error() string { return "aws route53: " + e.msg }

func newRoute53Error(msg string) route53Error { return route53Error{msg: msg} }
