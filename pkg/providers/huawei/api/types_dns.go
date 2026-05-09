package api

// DNS types for Huawei Cloud DNS service. The data model follows the public
// `dns.<region>.myhuaweicloud.com` REST API:
//
//   GET /v2/zones                       — list public zones (private zones live
//                                          on a separate path that is not
//                                          surfaced through cloudlist).
//   GET /v2/zones/{zone_id}/recordsets  — list record sets in a zone.
//
// Reference: Huawei Cloud DNS API doc set (zones & record_sets v2).

// ListZonesResponse is the typed shape of `GET /v2/zones`. Only public zones
// are listed at this path; private zones use a separate endpoint.
type ListZonesResponse struct {
	Links    *DNSLink  `json:"links"`
	Metadata DNSMeta   `json:"metadata"`
	Zones    []DNSZone `json:"zones"`
}

// DNSZone is the per-zone wire representation. `Name` is fully-qualified with
// a trailing dot ("example.com.") in DNS API responses; the cloudlist driver
// trims the dot before surfacing.
type DNSZone struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Description string `json:"description"`
	Status      string `json:"status"`
	ZoneType    string `json:"zone_type"`
	RecordNum   int64  `json:"record_num"`
	PoolID      string `json:"pool_id"`
	ProjectID   string `json:"project_id"`
}

// ListRecordSetsResponse is the typed shape of
// `GET /v2/zones/{zone_id}/recordsets`. Records is heterogeneous over record
// type — the wire format keeps every value as a string, with type-specific
// formatting (e.g. MX is "<preference> <exchange>").
type ListRecordSetsResponse struct {
	Links      *DNSLink    `json:"links"`
	Metadata   DNSMeta     `json:"metadata"`
	RecordSets []DNSRecord `json:"recordsets"`
}

type DNSRecord struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	TTL         int64    `json:"ttl"`
	Status      string   `json:"status"`
	Description string   `json:"description"`
	ZoneID      string   `json:"zone_id"`
	Records     []string `json:"records"`
}

type DNSLink struct {
	Self string `json:"self"`
	Next string `json:"next,omitempty"`
}

type DNSMeta struct {
	TotalCount int64 `json:"total_count"`
}
