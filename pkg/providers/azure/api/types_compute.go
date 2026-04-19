package api

const ComputeAPIVersion = "2022-08-01"

type VirtualMachine struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Location   string              `json:"location"`
	Status     string              `json:"status"`
	Properties VirtualMachineProps `json:"properties"`
}

type VirtualMachineProps struct {
	ProvisioningState string             `json:"provisioningState"`
	NetworkProfile    *VMNetworkProfile  `json:"networkProfile,omitempty"`
}

type VMNetworkProfile struct {
	NetworkInterfaces []VMNetworkInterfaceRef `json:"networkInterfaces"`
}

type VMNetworkInterfaceRef struct {
	ID string `json:"id"`
}

type ListVirtualMachinesResponse struct {
	Value    []VirtualMachine `json:"value"`
	NextLink string           `json:"nextLink"`
}
