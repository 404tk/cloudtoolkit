package replay

import (
	"fmt"
	"net/http"
	"time"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleListSQLServers serves the subscription-scoped Microsoft.Sql/servers
// list used by the cloudlist `database` asset. The fixture returns a small
// projection of demo SQL servers spread across resource groups; PATCHing a
// specific server is still routed via handleSQLServer at the resource-group
// scope.
func (t *transport) handleListSQLServers(req *http.Request, subscription string) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return armErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			fmt.Sprintf("method %s not supported on Microsoft.Sql/servers", req.Method)), nil
	}
	resp := struct {
		Value []azapi.SQLServer `json:"value"`
	}{}
	resp.Value = append(resp.Value, demoSQLServers(subscription)...)
	return jsonResponse(req, resp), nil
}

// handleListWorkspaces serves the subscription-scoped
// Microsoft.OperationalInsights/workspaces list used by the cloudlist `log`
// asset.
func (t *transport) handleListWorkspaces(req *http.Request, subscription string) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return armErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			fmt.Sprintf("method %s not supported on Microsoft.OperationalInsights/workspaces", req.Method)), nil
	}
	resp := struct {
		Value []azapi.Workspace `json:"value"`
	}{}
	resp.Value = append(resp.Value, demoWorkspaces(subscription)...)
	return jsonResponse(req, resp), nil
}

// handleCostManagementQuery serves Microsoft.CostManagement/query used by the
// cloudlist `balance` asset. With granularity=None the response carries a
// single row; the fixture surfaces a small constant total.
func (t *transport) handleCostManagementQuery(req *http.Request, subscription string) (*http.Response, error) {
	if req.Method != http.MethodPost {
		return armErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			fmt.Sprintf("method %s not supported on Microsoft.CostManagement/query", req.Method)), nil
	}
	resp := azapi.CostManagementQueryResponse{
		ID:   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.CostManagement/query", subscription),
		Name: "ctk-replay-cost",
		Type: "Microsoft.CostManagement/query",
		Properties: azapi.CostManagementQueryProperties{
			Columns: []azapi.CostManagementColumn{
				{Name: "Cost", Type: "Number"},
				{Name: "Currency", Type: "String"},
			},
			Rows: [][]any{
				{984.21, "USD"},
			},
		},
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func demoSQLServers(subscription string) []azapi.SQLServer {
	rg := "ctk-demo-rg"
	groups := resourceGroupsFor(subscription)
	if len(groups) > 0 {
		rg = groups[0]
	}
	return []azapi.SQLServer{
		{
			ID:       fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Sql/servers/ctk-demo-sql", subscription, rg),
			Name:     "ctk-demo-sql",
			Location: demoLocation,
			Properties: azapi.SQLServerProperties{
				AdministratorLogin:       "ctkadmin",
				FullyQualifiedDomainName: "ctk-demo-sql.database.windows.net",
				State:                    "Ready",
				Version:                  "12.0",
			},
		},
	}
}

func demoWorkspaces(subscription string) []azapi.Workspace {
	rg := "ctk-demo-rg"
	groups := resourceGroupsFor(subscription)
	if len(groups) > 0 {
		rg = groups[0]
	}
	created := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	return []azapi.Workspace{
		{
			ID:       fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.OperationalInsights/workspaces/ctk-demo-logs", subscription, rg),
			Name:     "ctk-demo-logs",
			Type:     "Microsoft.OperationalInsights/workspaces",
			Location: demoLocation,
			Properties: azapi.WorkspaceProps{
				CustomerID:        "00000000-0000-0000-0000-000000000010",
				ProvisioningState: "Succeeded",
				CreatedDate:       created,
				ModifiedDate:      created,
				RetentionInDays:   30,
			},
		},
	}
}
