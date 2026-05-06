package api

const DNSAPIVersion = "2018-05-01"

// DNSZone is the management-plane representation of a public Azure DNS zone
// (`Microsoft.Network/dnsZones`). Private DNS lives under
// `Microsoft.Network/privateDnsZones` with a separate API version; the demo
// surface only models public zones for now (CSPM signal value is highest on
// internet-facing zones).
type DNSZone struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Location   string         `json:"location"`
	Properties DNSZoneProps   `json:"properties"`
}

type DNSZoneProps struct {
	NumberOfRecordSets         int64    `json:"numberOfRecordSets"`
	MaxNumberOfRecordSets      int64    `json:"maxNumberOfRecordSets"`
	NameServers                []string `json:"nameServers"`
	ZoneType                   string   `json:"zoneType"`
}

// DNSRecordSet covers the union of record-set property shapes Azure returns
// in the `recordSets` list for a DNS zone. Only one of the typed slices is
// populated per record set, keyed by the trailing segment of `Type`
// (e.g. `Microsoft.Network/dnszones/A` → ARecords).
type DNSRecordSet struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Type       string             `json:"type"`
	Etag       string             `json:"etag"`
	Properties DNSRecordSetProps  `json:"properties"`
}

type DNSRecordSetProps struct {
	TTL          int64                `json:"TTL"`
	FQDN         string               `json:"fqdn"`
	ARecords     []DNSARecord         `json:"ARecords,omitempty"`
	AAAARecords  []DNSAAAARecord      `json:"AAAARecords,omitempty"`
	CNAMERecord  *DNSCNAMERecord      `json:"CNAMERecord,omitempty"`
	MXRecords    []DNSMXRecord        `json:"MXRecords,omitempty"`
	TXTRecords   []DNSTXTRecord       `json:"TXTRecords,omitempty"`
	NSRecords    []DNSNSRecord        `json:"NSRecords,omitempty"`
	SOARecord    *DNSSOARecord        `json:"SOARecord,omitempty"`
}

type DNSARecord struct {
	IPv4Address string `json:"ipv4Address"`
}

type DNSAAAARecord struct {
	IPv6Address string `json:"ipv6Address"`
}

type DNSCNAMERecord struct {
	CNAME string `json:"cname"`
}

type DNSMXRecord struct {
	Preference int64  `json:"preference"`
	Exchange   string `json:"exchange"`
}

type DNSTXTRecord struct {
	Value []string `json:"value"`
}

type DNSNSRecord struct {
	NSDName string `json:"nsdname"`
}

type DNSSOARecord struct {
	Email string `json:"email"`
	Host  string `json:"host"`
}
