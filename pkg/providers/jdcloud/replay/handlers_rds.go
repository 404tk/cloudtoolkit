package replay

import (
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleRDS serves the JDCloud RDS account lifecycle paths used by
// rds-account-check. Endpoints are pattern-inferred from JDCloud's regional
// REST convention.
func (t *transport) handleRDS(req *http.Request, _ []byte) (*http.Response, error) {
	path := req.URL.Path
	method := strings.ToUpper(req.Method)
	switch {
	case method == http.MethodPost && strings.HasSuffix(path, "/accounts"):
		instanceID := extractInstanceID(path)
		if instanceID == "" {
			return apiErrorResponse(req, http.StatusBadRequest, "InvalidPath", "malformed accounts path"), nil
		}
		t.addRDSAccount(instanceID, "ctkuser")
		resp := api.CreateRDSAccountResponse{RequestID: "req-replay-rds-create-account"}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case method == http.MethodDelete && strings.Contains(path, "/accounts/"):
		instanceID, account := splitAccountPath(path)
		if instanceID == "" || account == "" {
			return apiErrorResponse(req, http.StatusBadRequest, "InvalidPath", "malformed accounts path"), nil
		}
		if !t.removeRDSAccount(instanceID, account) {
			return apiErrorResponse(req, http.StatusNotFound, "ResourceNotFound", "account not found"), nil
		}
		resp := api.DeleteRDSAccountResponse{RequestID: "req-replay-rds-delete-account"}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case method == http.MethodGet && strings.HasSuffix(path, "/accounts"):
		instanceID := extractInstanceID(path)
		resp := api.DescribeRDSAccountsResponse{RequestID: "req-replay-rds-describe-accounts"}
		for _, name := range t.snapshotRDSAccounts(instanceID) {
			resp.Result.Accounts = append(resp.Result.Accounts, api.RDSAccount{
				AccountName:   name,
				AccountStatus: "Available",
				AccountType:   "Normal",
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "InvalidPath",
		"unsupported rds path: "+path), nil
}

// extractInstanceID parses `/v1/regions/<region>/instances/<id>/accounts`.
func extractInstanceID(path string) string {
	idx := strings.Index(path, "/instances/")
	if idx < 0 {
		return ""
	}
	rest := path[idx+len("/instances/"):]
	end := strings.Index(rest, "/")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// splitAccountPath parses `.../instances/<id>/accounts/<account>`.
func splitAccountPath(path string) (string, string) {
	instanceID := extractInstanceID(path)
	if instanceID == "" {
		return "", ""
	}
	idx := strings.LastIndex(path, "/accounts/")
	if idx < 0 {
		return instanceID, ""
	}
	account := strings.TrimSpace(path[idx+len("/accounts/"):])
	return instanceID, account
}
