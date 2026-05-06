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

// RDSDBUser models the RDS MySQL user resource. The control-plane endpoints
// share this shape across engines (MySQL, PostgreSQL); per-engine paths
// differ but the request/response payloads are equivalent.
type RDSDBUser struct {
	Name    string `json:"name"`
	Host    string `json:"host,omitempty"`
	Comment string `json:"comment,omitempty"`
	State   string `json:"state,omitempty"`
}

type ListRDSDBUsersResponse struct {
	Users      []RDSDBUser `json:"users"`
	TotalCount *int32      `json:"total_count"`
}

type CreateRDSDBUserRequest struct {
	Name     string   `json:"name"`
	Password string   `json:"password"`
	Hosts    []string `json:"hosts,omitempty"`
	Comment  string   `json:"comment,omitempty"`
}

type CreateRDSDBUserResponse struct {
	Resp string `json:"resp,omitempty"`
}

type DeleteRDSDBUserResponse struct {
	Resp string `json:"resp,omitempty"`
}
