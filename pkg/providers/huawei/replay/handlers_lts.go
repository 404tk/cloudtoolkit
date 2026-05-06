package replay

import (
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleLTS serves LTS `ListLogGroups` (`GET /v2/<project_id>/groups`) used by
// the cloudlist `log` asset. The project_id is resolved upstream via the IAM
// replay; here we accept any project_id and return a small fixture set.
func (t *transport) handleLTS(req *http.Request, _ string) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "LTS.0001",
			"unsupported lts method: "+req.Method), nil
	}
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "v2" || parts[2] != "groups" {
		return apiErrorResponse(req, http.StatusNotFound, "LTS.0001",
			"unsupported lts path: "+req.URL.Path), nil
	}
	resp := api.ListLogGroupsResponse{
		LogGroups: []api.LTSLogGroup{
			{LogGroupID: "lg-ctk-demo-app", LogGroupName: "ctk-demo-app", CreationTime: 1745020800000, TTLInDays: 14},
			{LogGroupID: "lg-ctk-demo-audit", LogGroupName: "ctk-demo-audit", CreationTime: 1745107200000, TTLInDays: 90},
		},
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}
