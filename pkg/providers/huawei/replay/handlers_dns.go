package replay

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// dnsZoneFixture is the small Huawei DNS state the replay needs: zone meta
// + flat record list. One zone produces one cloudlist `domain` asset, which
// drives the demo flow end-to-end without dragging real DNS infrastructure.
type huaweiDNSZoneFixture struct {
	ID      string
	Name    string // FQDN with trailing dot, mirroring the real wire format
	Records []huaweiDNSRecordFixture
}

type huaweiDNSRecordFixture struct {
	ID     string
	Name   string
	Type   string
	TTL    int64
	Status string
	Values []string
}

var demoHuaweiDNSZones = []huaweiDNSZoneFixture{
	{
		ID:   "z-ctk-public-1",
		Name: "ctk-demo.example.com.",
		Records: []huaweiDNSRecordFixture{
			{ID: "r-a", Name: "ctk-demo.example.com.", Type: "A", TTL: 300, Status: "ACTIVE",
				Values: []string{"203.0.113.20", "203.0.113.21"}},
			{ID: "r-cname", Name: "www.ctk-demo.example.com.", Type: "CNAME", TTL: 60, Status: "ACTIVE",
				Values: []string{"ctk-demo.example.com."}},
			{ID: "r-mx", Name: "ctk-demo.example.com.", Type: "MX", TTL: 300, Status: "ACTIVE",
				Values: []string{"10 mx1.ctk-demo.example.com."}},
			// NS is included intentionally; the driver should filter it.
			{ID: "r-ns", Name: "ctk-demo.example.com.", Type: "NS", TTL: 172800, Status: "ACTIVE",
				Values: []string{"ns1.example.com."}},
		},
	},
}

func findHuaweiDNSZone(id string) (huaweiDNSZoneFixture, bool) {
	for _, z := range demoHuaweiDNSZones {
		if z.ID == id {
			return z, true
		}
	}
	return huaweiDNSZoneFixture{}, false
}

func (t *transport) handleDNS(req *http.Request, region string) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "DNS.0001",
			fmt.Sprintf("dns replay expects GET, got %s", req.Method)), nil
	}
	_ = region // public zones are account-scoped; region is irrelevant here

	path := strings.Trim(req.URL.Path, "/")
	parts := strings.Split(path, "/")

	switch {
	case len(parts) == 2 && parts[0] == "v2" && parts[1] == "zones":
		resp := api.ListZonesResponse{Metadata: api.DNSMeta{TotalCount: int64(len(demoHuaweiDNSZones))}}
		for _, z := range demoHuaweiDNSZones {
			resp.Zones = append(resp.Zones, api.DNSZone{
				ID:        z.ID,
				Name:      z.Name,
				Status:    "ACTIVE",
				ZoneType:  "public",
				RecordNum: int64(len(z.Records)),
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil

	case len(parts) == 4 && parts[0] == "v2" && parts[1] == "zones" && parts[3] == "recordsets":
		zone, ok := findHuaweiDNSZone(parts[2])
		if !ok {
			return apiErrorResponse(req, http.StatusNotFound, "DNS.0301",
				fmt.Sprintf("zone %s not found", parts[2])), nil
		}
		resp := api.ListRecordSetsResponse{Metadata: api.DNSMeta{TotalCount: int64(len(zone.Records))}}
		for _, r := range zone.Records {
			resp.RecordSets = append(resp.RecordSets, api.DNSRecord{
				ID:      r.ID,
				Name:    r.Name,
				Type:    r.Type,
				TTL:     r.TTL,
				Status:  r.Status,
				ZoneID:  zone.ID,
				Records: append([]string(nil), r.Values...),
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "DNS.0001",
		fmt.Sprintf("unsupported dns path: %s", req.URL.Path)), nil
}
