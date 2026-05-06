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

// InstanceMetadataItem mirrors GCE's instance metadata key/value pair.
type InstanceMetadataItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// InstanceMetadata is the wrapper Compute Engine returns; SetMetadata requires
// the fingerprint to detect concurrent edits.
type InstanceMetadata struct {
	Fingerprint string                 `json:"fingerprint"`
	Items       []InstanceMetadataItem `json:"items"`
	Kind        string                 `json:"kind,omitempty"`
}

// InstanceWithMetadata is the subset of `compute.instances.get` we need to
// run the metadata startup-script + reboot path. Compute Engine's full
// Instance shape is much larger; we only project metadata + the basics.
type InstanceWithMetadata struct {
	Name     string           `json:"name"`
	Zone     string           `json:"zone"`
	Status   string           `json:"status"`
	Metadata InstanceMetadata `json:"metadata"`
}

// ComputeOperation is the LRO surface returned by setMetadata / reset.
type ComputeOperation struct {
	Name       string `json:"name"`
	Zone       string `json:"zone"`
	Status     string `json:"status"`
	OperationType string `json:"operationType"`
	TargetLink string `json:"targetLink"`
	Error      *ComputeOperationError `json:"error,omitempty"`
}

type ComputeOperationError struct {
	Errors []ComputeOperationErrorItem `json:"errors"`
}

type ComputeOperationErrorItem struct {
	Code     string `json:"code"`
	Location string `json:"location"`
	Message  string `json:"message"`
}
