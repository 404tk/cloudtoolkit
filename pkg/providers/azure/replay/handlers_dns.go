package replay

import (
	"fmt"
	"net/http"
	"strings"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// dnsZoneFixture mirrors the small slice of Azure DNS state the replay
// transport needs: the zone metadata plus a flat list of record sets keyed
// by `<recordType>/<recordName>` (the Azure ARM URL fragment). Each fixture
// produces one cloudlist `domain` asset, which is enough to drive the
// validation flow end-to-end without dragging in a real DNS shape.
type dnsZoneFixture struct {
	Name          string
	ResourceGroup string
	Records       []dnsRecordFixture
}

type dnsRecordFixture struct {
	Name string // record-set name relative to the zone ("@", "www", "mail")
	Type string // A / AAAA / CNAME / MX / TXT / NS
	TTL  int64
	// One of the typed slices below is filled, mirroring Azure ARM's record
	// property union.
	A     []string
	AAAA  []string
	CNAME string
	MX    []dnsMXFixture
	TXT   []string
	NS    []string
}

type dnsMXFixture struct {
	Preference int64
	Exchange   string
}

var demoAzureDNSZones = []dnsZoneFixture{
	{
		Name:          "ctk-demo.example.com",
		ResourceGroup: demoResourceGroup,
		Records: []dnsRecordFixture{
			{Name: "@", Type: "A", TTL: 300, A: []string{"203.0.113.20", "203.0.113.21"}},
			{Name: "www", Type: "CNAME", TTL: 60, CNAME: "ctk-demo.example.com."},
			{Name: "mail", Type: "MX", TTL: 300, MX: []dnsMXFixture{{Preference: 10, Exchange: "mx1.ctk-demo.example.com."}}},
			// NS is intentionally included — the driver filters it, which the
			// audit can verify (no NS record should appear in cloudlist output).
			{Name: "@", Type: "NS", TTL: 172800, NS: []string{"ns1-01.azure-dns.com."}},
		},
	},
}

func findDNSZone(name string) (dnsZoneFixture, bool) {
	for _, z := range demoAzureDNSZones {
		if z.Name == name {
			return z, true
		}
	}
	return dnsZoneFixture{}, false
}

// handleListDNSZones services the subscription-scoped list at
//
//	/subscriptions/{sub}/providers/Microsoft.Network/dnsZones
func (t *transport) handleListDNSZones(req *http.Request, subscription string) (*http.Response, error) {
	resp := dnsListZonesResponse{}
	for _, z := range demoAzureDNSZones {
		resp.Value = append(resp.Value, dnsZoneWire{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/dnsZones/%s",
				subscription, z.ResourceGroup, z.Name),
			Name:     z.Name,
			Type:     "Microsoft.Network/dnszones",
			Location: "global",
			Properties: dnsZonePropsWire{
				NumberOfRecordSets: int64(len(z.Records)),
				ZoneType:           "Public",
			},
		})
	}
	return jsonResponse(req, resp), nil
}

// handleDNSZoneScoped services the resource-group-scoped path
//
//	/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/dnsZones/{zone}[/recordsets]
func (t *transport) handleDNSZoneScoped(req *http.Request, subscription, group string, rest []string) (*http.Response, error) {
	if len(rest) == 0 {
		return armErrorResponse(req, http.StatusNotFound, "InvalidPath", "dnsZones path missing zone name"), nil
	}
	zoneName := rest[0]
	zone, ok := findDNSZone(zoneName)
	if !ok {
		return armErrorResponse(req, http.StatusNotFound, "NoSuchHostedZone",
			fmt.Sprintf("dns zone %s not found in resource group %s", zoneName, group)), nil
	}
	if zone.ResourceGroup != group {
		return armErrorResponse(req, http.StatusNotFound, "NoSuchHostedZone",
			fmt.Sprintf("dns zone %s not in resource group %s", zoneName, group)), nil
	}
	if len(rest) >= 2 && strings.EqualFold(rest[1], "recordsets") {
		return handleDNSRecordSets(req, subscription, group, zone)
	}
	return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
		fmt.Sprintf("unsupported dnsZones subpath: %v", rest)), nil
}

func handleDNSRecordSets(req *http.Request, subscription, group string, zone dnsZoneFixture) (*http.Response, error) {
	resp := dnsListRecordSetsResponse{}
	for _, r := range zone.Records {
		wire := dnsRecordSetWire{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/dnsZones/%s/%s/%s",
				subscription, group, zone.Name, r.Type, r.Name),
			Name: r.Name,
			Type: "Microsoft.Network/dnszones/" + r.Type,
			Properties: dnsRecordSetPropsWire{
				TTL:  r.TTL,
				FQDN: dnsFQDN(zone.Name, r.Name),
			},
		}
		switch r.Type {
		case "A":
			for _, ip := range r.A {
				wire.Properties.ARecords = append(wire.Properties.ARecords, dnsAWire{IPv4: ip})
			}
		case "AAAA":
			for _, ip := range r.AAAA {
				wire.Properties.AAAARecords = append(wire.Properties.AAAARecords, dnsAAAAWire{IPv6: ip})
			}
		case "CNAME":
			if r.CNAME != "" {
				wire.Properties.CNAMERecord = &dnsCNAMEWire{CNAME: r.CNAME}
			}
		case "MX":
			for _, m := range r.MX {
				wire.Properties.MXRecords = append(wire.Properties.MXRecords, dnsMXWire{Preference: m.Preference, Exchange: m.Exchange})
			}
		case "TXT":
			for _, v := range r.TXT {
				wire.Properties.TXTRecords = append(wire.Properties.TXTRecords, dnsTXTWire{Value: []string{v}})
			}
		case "NS":
			for _, ns := range r.NS {
				wire.Properties.NSRecords = append(wire.Properties.NSRecords, dnsNSWire{NSDName: ns})
			}
		}
		resp.Value = append(resp.Value, wire)
	}
	return jsonResponse(req, resp), nil
}

func dnsFQDN(zone, name string) string {
	if name == "" || name == "@" {
		return zone + "."
	}
	return name + "." + zone + "."
}

// JSON wire types for the DNS replay handlers. The driver decodes into
// azapi.DNSZone / azapi.DNSRecordSet which use the same shape but with
// schema-friendly Go field names; using independent local types keeps the
// replay package free of dependencies on the driver internals.

type dnsListZonesResponse struct {
	Value    []dnsZoneWire `json:"value"`
	NextLink string        `json:"nextLink,omitempty"`
}

type dnsZoneWire struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Type       string           `json:"type"`
	Location   string           `json:"location"`
	Properties dnsZonePropsWire `json:"properties"`
}

type dnsZonePropsWire struct {
	NumberOfRecordSets int64    `json:"numberOfRecordSets"`
	NameServers        []string `json:"nameServers,omitempty"`
	ZoneType           string   `json:"zoneType"`
}

type dnsListRecordSetsResponse struct {
	Value    []dnsRecordSetWire `json:"value"`
	NextLink string             `json:"nextLink,omitempty"`
}

type dnsRecordSetWire struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Type       string                `json:"type"`
	Properties dnsRecordSetPropsWire `json:"properties"`
}

type dnsRecordSetPropsWire struct {
	TTL          int64           `json:"TTL"`
	FQDN         string          `json:"fqdn"`
	ARecords     []dnsAWire      `json:"ARecords,omitempty"`
	AAAARecords  []dnsAAAAWire   `json:"AAAARecords,omitempty"`
	CNAMERecord  *dnsCNAMEWire   `json:"CNAMERecord,omitempty"`
	MXRecords    []dnsMXWire     `json:"MXRecords,omitempty"`
	TXTRecords   []dnsTXTWire    `json:"TXTRecords,omitempty"`
	NSRecords    []dnsNSWire     `json:"NSRecords,omitempty"`
}

type dnsAWire struct {
	IPv4 string `json:"ipv4Address"`
}

type dnsAAAAWire struct {
	IPv6 string `json:"ipv6Address"`
}

type dnsCNAMEWire struct {
	CNAME string `json:"cname"`
}

type dnsMXWire struct {
	Preference int64  `json:"preference"`
	Exchange   string `json:"exchange"`
}

type dnsTXTWire struct {
	Value []string `json:"value"`
}

type dnsNSWire struct {
	NSDName string `json:"nsdname"`
}

// keep azapi/demoreplay import live so future handler edits stay consistent.
var _ = azapi.DNSAPIVersion
var _ = demoreplay.AuthOK
