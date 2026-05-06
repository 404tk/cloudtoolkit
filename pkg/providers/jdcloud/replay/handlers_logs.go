package replay

import (
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleLogs serves the cloudlist `log` asset endpoint
// `/v1/regions/<region>/logTopics:describe`. Pattern-inferred — see
// pkg/providers/jdcloud/api/types_logs.go.
func (t *transport) handleLogs(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet || !strings.HasSuffix(req.URL.Path, "/logTopics:describe") {
		return apiErrorResponse(req, http.StatusNotFound, "InvalidPath",
			"unsupported logs path: "+req.URL.Path), nil
	}
	resp := api.DescribeLogTopicsResponse{RequestID: "req-replay-logs-describe-topics"}
	resp.Result.Topics = []api.LogTopic{
		{
			LogTopicID:   "topic-ctk-demo-app",
			LogTopicName: "ctk-demo-app",
			Description:  "ctk demo application logs",
			LogSetID:     "set-ctk-demo",
			LogSetName:   "ctk-demo",
			CreateTime:   "2026-04-01T08:00:00Z",
			UpdateTime:   "2026-04-22T09:00:00Z",
		},
		{
			LogTopicID:   "topic-ctk-demo-audit",
			LogTopicName: "ctk-demo-audit",
			Description:  "ctk demo audit pipeline",
			LogSetID:     "set-ctk-demo",
			LogSetName:   "ctk-demo",
			CreateTime:   "2026-03-15T08:00:00Z",
			UpdateTime:   "2026-04-22T09:00:00Z",
		},
	}
	resp.Result.TotalCount = len(resp.Result.Topics)
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}
