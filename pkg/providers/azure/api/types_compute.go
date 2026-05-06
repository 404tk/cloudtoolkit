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

// RunCommandInput models the request body for `virtualMachines/runCommand`.
// `commandId` selects a built-in script: `RunShellScript` (Linux) or
// `RunPowerShellScript` (Windows). `script` holds the command lines.
type RunCommandInput struct {
	CommandID string   `json:"commandId"`
	Script    []string `json:"script"`
}

// RunCommandResult is the synchronous response payload. ARM may return 202
// with a Location header for long-running ops; the simple cases return the
// inline result.
type RunCommandResult struct {
	Value []RunCommandInstanceView `json:"value"`
}

type RunCommandInstanceView struct {
	Code          string `json:"code"`
	Level         string `json:"level"`
	DisplayStatus string `json:"displayStatus"`
	Message       string `json:"message"`
}
