package replay

import (
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleLogs serves the cloudlist `log` asset endpoint
// `/v1/regions/<region>/logsets`.
func (t *transport) handleLogs(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet || !strings.HasSuffix(req.URL.Path, "/logsets") {
		return apiErrorResponse(req, http.StatusNotFound, "InvalidPath",
			"unsupported logs path: "+req.URL.Path), nil
	}
	resp := api.DescribeLogsetsResponse{RequestID: "req-replay-logs-describe-logsets"}
	resp.Result.Data = []api.LogsetEnd{
		{
			UID:         "set-ctk-demo-app",
			Name:        "ctk-demo-app",
			Description: "ctk demo application logset",
			HasTopic:    true,
			Region:      "cn-north-1",
			CreateTime:  "2026-04-01T08:00:00Z",
			LifeCycle:   30,
		},
		{
			UID:         "set-ctk-demo-audit",
			Name:        "ctk-demo-audit",
			Description: "ctk demo audit logset",
			HasTopic:    true,
			Region:      "cn-north-1",
			CreateTime:  "2026-03-15T08:00:00Z",
			LifeCycle:   90,
		},
	}
	resp.Result.NumberRecords = int64(len(resp.Result.Data))
	resp.Result.PageNumber = 1
	resp.Result.PageSize = int64(len(resp.Result.Data))
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}
