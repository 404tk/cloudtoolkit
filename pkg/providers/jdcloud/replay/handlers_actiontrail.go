package replay

import (
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleActionTrail serves the JDCloud audit-log lookup used by event-check.
// The path is pattern-inferred — `/v1/regions/<region>/events:lookup`.
func (t *transport) handleActionTrail(req *http.Request) (*http.Response, error) {
	path := req.URL.Path
	if req.Method != http.MethodGet || !strings.HasSuffix(path, ":lookup") {
		return apiErrorResponse(req, http.StatusNotFound, "InvalidAction",
			"unsupported actiontrail path: "+path), nil
	}
	resp := api.DescribeActionTrailEventsResponse{RequestID: "req-replay-actiontrail-lookup"}
	resp.Result.Events = demoActionTrailEvents()
	resp.Result.TotalCount = len(resp.Result.Events)
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func demoActionTrailEvents() []api.ActionTrailEvent {
	return []api.ActionTrailEvent{
		{
			EventID:         "jdc-evt-0001",
			EventName:       "CreateSubUser",
			EventTime:       "2026-04-22T09:11:00Z",
			EventSource:     "iam.jdcloud-api.com",
			UserName:        "admin",
			SourceIPAddress: "203.0.113.62",
			Region:          "cn-north-1",
			Status:          "Success",
			AccessKey:       "JDC_AKLT_ACTKDEMO000",
			ResourceName:    "subUser/audit",
			ResourceType:    "iam:SubUser",
		},
		{
			EventID:         "jdc-evt-0002",
			EventName:       "PutBucketAcl",
			EventTime:       "2026-04-22T09:14:30Z",
			EventSource:     "oss.jdcloud-api.com",
			UserName:        "admin",
			SourceIPAddress: "203.0.113.62",
			Region:          "cn-north-1",
			Status:          "Success",
			AccessKey:       "JDC_AKLT_ACTKDEMO000",
			ResourceName:    "ctk-jdcloud-public",
			ResourceType:    "oss:Bucket",
		},
		{
			EventID:         "jdc-evt-0003",
			EventName:       "DeleteAccessKey",
			EventTime:       "2026-04-22T09:18:12Z",
			EventSource:     "iam.jdcloud-api.com",
			UserName:        "admin",
			SourceIPAddress: "203.0.113.62",
			Region:          "cn-north-1",
			Status:          "Failed",
			AccessKey:       "JDC_AKLT_ACTKDEMO000",
			ResourceName:    "subUser/audit",
			ResourceType:    "iam:AccessKey",
		},
	}
}
