package replay

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// Route53 is global; the data plane host is `route53.amazonaws.com` (or
// `route53.amazonaws.com.cn`). Path patterns in flight:
//   - GET /2013-04-01/hostedzone
//   - GET /2013-04-01/hostedzone/{Id}/rrset
//
// The replay deliberately keeps the surface small — list zones + list records
// is enough to drive the demo `cloudlist` flow end-to-end.

func isRoute53Host(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "route53.amazonaws.com" || host == "route53.amazonaws.com.cn"
}

type route53HostedZoneFixture struct {
	ID            string
	Name          string
	PrivateZone   bool
	Records       []route53RecordFixture
	Comment       string
	CallerRefence string
}

type route53RecordFixture struct {
	Name    string
	Type    string
	TTL     int64
	Values  []string
	IsAlias bool
}

var demoRoute53Zones = []route53HostedZoneFixture{
	{
		ID:            "Z2CTKDEMOPUBLIC0001",
		Name:          "ctk-demo.example.com.",
		PrivateZone:   false,
		Comment:       "ctk demo public zone",
		CallerRefence: "ctk-demo-public",
		Records: []route53RecordFixture{
			{Name: "ctk-demo.example.com.", Type: "A", TTL: 300, Values: []string{"198.51.100.10"}},
			{Name: "www.ctk-demo.example.com.", Type: "CNAME", TTL: 60, Values: []string{"ctk-demo.example.com."}},
			{Name: "ctk-demo.example.com.", Type: "MX", TTL: 300, Values: []string{"10 mx1.ctk-demo.example.com."}},
			{Name: "ctk-demo.example.com.", Type: "NS", TTL: 172800, Values: []string{"ns-1.ctk-demo.example.com."}},
		},
	},
	{
		ID:            "Z2CTKDEMOPRIVATE001",
		Name:          "internal.ctk-demo.example.",
		PrivateZone:   true,
		Comment:       "ctk demo private zone",
		CallerRefence: "ctk-demo-private",
		Records: []route53RecordFixture{
			{Name: "db.internal.ctk-demo.example.", Type: "A", TTL: 60, Values: []string{"10.0.0.10"}},
		},
	},
}

func findRoute53Zone(id string) (route53HostedZoneFixture, bool) {
	id = strings.TrimSpace(id)
	for _, z := range demoRoute53Zones {
		if z.ID == id {
			return z, true
		}
	}
	return route53HostedZoneFixture{}, false
}

func (t *transport) handleRoute53(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed", fmt.Sprintf("route53 replay only supports GET, got %s", req.Method)), nil
	}
	path := strings.TrimPrefix(req.URL.Path, "/")
	const apiVersion = "2013-04-01"
	if !strings.HasPrefix(path, apiVersion+"/hostedzone") {
		return apiErrorResponse(req, http.StatusNotFound, "InvalidRoute", fmt.Sprintf("unsupported route53 path: %s", req.URL.Path)), nil
	}
	rest := strings.TrimPrefix(path, apiVersion+"/hostedzone")
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		// list hosted zones
		resp := route53ListHostedZonesResponse{IsTruncated: false}
		for _, z := range demoRoute53Zones {
			resp.HostedZones = append(resp.HostedZones, route53HostedZoneWire{
				ID:              "/hostedzone/" + z.ID,
				Name:            z.Name,
				CallerReference: z.CallerRefence,
				Config: route53HostedZoneConfigWire{
					Comment:     z.Comment,
					PrivateZone: z.PrivateZone,
				},
				ResourceRecordSetCount: int64(len(z.Records)),
			})
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	}
	// expect "<id>/rrset"
	parts := strings.SplitN(rest, "/", 2)
	zoneID := parts[0]
	zone, ok := findRoute53Zone(zoneID)
	if !ok {
		return apiErrorResponse(req, http.StatusNotFound, "NoSuchHostedZone", fmt.Sprintf("hosted zone %s not found", zoneID)), nil
	}
	if len(parts) < 2 || parts[1] != "rrset" {
		return apiErrorResponse(req, http.StatusNotFound, "InvalidRoute", fmt.Sprintf("unsupported route53 sub-path: %s", parts[1])), nil
	}
	resp := route53ListRecordSetsResponse{IsTruncated: false}
	for _, r := range zone.Records {
		wire := route53RRSetWire{Name: r.Name, Type: r.Type, TTL: r.TTL}
		for _, v := range r.Values {
			wire.ResourceRecords = append(wire.ResourceRecords, route53RRWire{Value: v})
		}
		resp.ResourceRecordSets = append(resp.ResourceRecordSets, wire)
	}
	return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
}

type route53ListHostedZonesResponse struct {
	XMLName     xml.Name                `xml:"ListHostedZonesResponse"`
	HostedZones []route53HostedZoneWire `xml:"HostedZones>HostedZone"`
	IsTruncated bool                    `xml:"IsTruncated"`
	MaxItems    string                  `xml:"MaxItems,omitempty"`
}

type route53HostedZoneWire struct {
	ID                     string                       `xml:"Id"`
	Name                   string                       `xml:"Name"`
	CallerReference        string                       `xml:"CallerReference"`
	Config                 route53HostedZoneConfigWire  `xml:"Config"`
	ResourceRecordSetCount int64                        `xml:"ResourceRecordSetCount"`
}

type route53HostedZoneConfigWire struct {
	Comment     string `xml:"Comment"`
	PrivateZone bool   `xml:"PrivateZone"`
}

type route53ListRecordSetsResponse struct {
	XMLName            xml.Name             `xml:"ListResourceRecordSetsResponse"`
	ResourceRecordSets []route53RRSetWire   `xml:"ResourceRecordSets>ResourceRecordSet"`
	IsTruncated        bool                 `xml:"IsTruncated"`
	MaxItems           string               `xml:"MaxItems,omitempty"`
}

type route53RRSetWire struct {
	Name            string         `xml:"Name"`
	Type            string         `xml:"Type"`
	TTL             int64          `xml:"TTL,omitempty"`
	ResourceRecords []route53RRWire `xml:"ResourceRecords>ResourceRecord"`
}

type route53RRWire struct {
	Value string `xml:"Value"`
}
