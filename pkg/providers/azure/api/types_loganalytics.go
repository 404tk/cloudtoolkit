package api

const OperationalInsightsAPIVersion = "2022-10-01"

// Workspace is the management-plane representation of a Log Analytics
// workspace (`Microsoft.OperationalInsights/workspaces`). It is the closest
// equivalent to the cloudlist `log` asset on Azure: a workspace is the
// container that aggregates logs from connected resources.
type Workspace struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Location   string         `json:"location"`
	Properties WorkspaceProps `json:"properties"`
}

type WorkspaceProps struct {
	CustomerID        string `json:"customerId"`
	ProvisioningState string `json:"provisioningState"`
	CreatedDate       string `json:"createdDate"`
	ModifiedDate      string `json:"modifiedDate"`
	RetentionInDays   int64  `json:"retentionInDays"`
	Sku               *struct {
		Name string `json:"name"`
	} `json:"sku,omitempty"`
}
