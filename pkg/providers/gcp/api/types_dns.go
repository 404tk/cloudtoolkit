package api

const DNSBaseURL = "https://dns.googleapis.com"

type ManagedZone struct {
	Name    string `json:"name"`
	DNSName string `json:"dnsName"`
}

type RRSet struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	RRDatas []string `json:"rrdatas"`
}

type ListManagedZonesResponse struct {
	ManagedZones  []ManagedZone `json:"managedZones"`
	NextPageToken string        `json:"nextPageToken"`
}

type ListRRSetsResponse struct {
	RRSets        []RRSet `json:"rrsets"`
	NextPageToken string  `json:"nextPageToken"`
}
