package api

const ComputeBaseURL = "https://compute.googleapis.com"

type Zone struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Instance struct {
	Hostname          string             `json:"hostname"`
	Name              string             `json:"name"`
	Zone              string             `json:"zone"`
	Status            string             `json:"status"`
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces"`
}

type NetworkInterface struct {
	NetworkIP     string         `json:"networkIP"`
	AccessConfigs []AccessConfig `json:"accessConfigs"`
}

type AccessConfig struct {
	NatIP string `json:"natIP"`
}

type ListZonesResponse struct {
	Items         []Zone `json:"items"`
	NextPageToken string `json:"nextPageToken"`
}

type ListInstancesResponse struct {
	Items         []Instance `json:"items"`
	NextPageToken string     `json:"nextPageToken"`
}
