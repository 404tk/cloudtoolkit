package api

type ListRDSInstancesResponse struct {
	Instances  []RDSInstance `json:"instances"`
	TotalCount *int32        `json:"total_count"`
}

type RDSInstance struct {
	ID         string        `json:"id"`
	Region     string        `json:"region"`
	Port       int32         `json:"port"`
	PublicIPs  []string      `json:"public_ips"`
	PrivateIPs []string      `json:"private_ips"`
	Datastore  *RDSDatastore `json:"datastore"`
}

type RDSDatastore struct {
	Type    string `json:"type"`
	Version string `json:"version"`
}
