package replay

import (
	"fmt"
	"net/http"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
)

// handleCLS serves the cloudlist `log` asset action `DescribeLogsets`.
func (t *transport) handleCLS(req *http.Request, action string) (*http.Response, error) {
	if action != "DescribeLogsets" {
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound",
			fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
	resp := api.DescribeLogsetsResponse{}
	resp.Response.RequestID = "req-replay-cls-describe-logsets"
	logsets := []api.CLSLogset{
		{
			LogsetID:   stringPtr("ls-ctk-demo-app"),
			LogsetName: stringPtr("ctk-demo-app"),
			CreateTime: stringPtr("2026-04-01 08:00:00"),
			TopicCount: uint64Ptr(2),
		},
		{
			LogsetID:   stringPtr("ls-ctk-demo-audit"),
			LogsetName: stringPtr("ctk-demo-audit"),
			CreateTime: stringPtr("2026-03-15 08:00:00"),
			TopicCount: uint64Ptr(1),
		},
	}
	resp.Response.Logsets = logsets
	total := uint64(len(logsets))
	resp.Response.TotalCount = &total
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}
