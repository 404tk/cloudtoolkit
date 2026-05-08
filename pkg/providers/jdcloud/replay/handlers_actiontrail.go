package replay

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleActionTrail serves the JDCloud audit-log lookup used by event-check.
// The replay path mirrors `/v1/regions/<region>/events`.
func (t *transport) handleActionTrail(req *http.Request, body []byte) (*http.Response, error) {
	path := req.URL.Path
	if req.Method != http.MethodPost || path != "/v1/regions/"+demoRegion+"/events" {
		return apiErrorResponse(req, http.StatusNotFound, "InvalidAction",
			"unsupported audittrail path: "+path), nil
	}
	lookup := lookupEventsRequestBody{
		PageSize:   len(demoActionTrailEvents()),
		PageNumber: 1,
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &lookup); err != nil {
			return apiErrorResponse(req, http.StatusBadRequest, "InvalidParameter",
				"invalid audittrail lookup body"), nil
		}
	}
	resp := api.DescribeActionTrailEventsResponse{RequestID: "req-replay-actiontrail-lookup"}
	events := filterActionTrailEvents(demoActionTrailEvents(), lookup.LookupAttributes)
	pageSize := lookup.PageSize
	pageNumber := lookup.PageNumber
	if pageSize <= 0 {
		pageSize = len(events)
	}
	start := (pageNumber - 1) * pageSize
	if start < 0 || start >= len(events) {
		resp.Result.Events = []api.ActionTrailEvent{}
	} else {
		end := start + pageSize
		if end > len(events) {
			end = len(events)
		}
		resp.Result.Events = events[start:end]
	}
	resp.Result.PageNumber = pageNumber
	resp.Result.PageSize = pageSize
	resp.Result.TotalNumber = int64(len(events))
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

type lookupEventsRequestBody struct {
	RegionID         string `json:"regionId"`
	PageSize         int    `json:"pageSize"`
	PageNumber       int    `json:"pageNumber"`
	LookupAttributes string `json:"lookupAttributes"`
}

func filterActionTrailEvents(events []api.ActionTrailEvent, lookupAttributes string) []api.ActionTrailEvent {
	lookupAttributes = strings.TrimSpace(lookupAttributes)
	if lookupAttributes == "" {
		return events
	}
	var attrs map[string]string
	if err := json.Unmarshal([]byte(lookupAttributes), &attrs); err != nil {
		return events
	}
	accessKeyID := strings.TrimSpace(attrs["accessKeyId"])
	if accessKeyID == "" {
		return events
	}
	out := make([]api.ActionTrailEvent, 0, len(events))
	for _, event := range events {
		if event.AccessKeyID == accessKeyID {
			out = append(out, event)
		}
	}
	return out
}

func demoActionTrailEvents() []api.ActionTrailEvent {
	return []api.ActionTrailEvent{
		{
			EventID:     "jdc-evt-0001",
			EventName:   "CreateSubUser",
			EventTime:   api.ActionTrailTimestamp(1776858660),
			EventSource: "iam.jdcloud-api.com",
			IP:          "203.0.113.62",
			Region:      "cn-north-1",
			AccessKeyID: demoCredentials.AccessKey,
			Resources: []api.ActionTrailResource{
				{ResourceName: "subUser/audit", ResourceType: "iam:SubUser"},
			},
		},
		{
			EventID:     "jdc-evt-0002",
			EventName:   "PutBucketAcl",
			EventTime:   api.ActionTrailTimestamp(1776858870),
			EventSource: "oss.jdcloud-api.com",
			IP:          "203.0.113.62",
			Region:      "cn-north-1",
			AccessKeyID: demoCredentials.AccessKey,
			Resources: []api.ActionTrailResource{
				{ResourceName: "ctk-jdcloud-public", ResourceType: "oss:Bucket"},
			},
		},
		{
			EventID:      "jdc-evt-0003",
			EventName:    "DeleteAccessKey",
			EventTime:    api.ActionTrailTimestamp(1776859092),
			EventSource:  "iam.jdcloud-api.com",
			IP:           "203.0.113.62",
			Region:       "cn-north-1",
			ErrorCode:    "AccessDenied",
			ErrorMessage: "permission denied",
			AccessKeyID:  demoCredentials.AccessKey,
			Resources: []api.ActionTrailResource{
				{ResourceName: "subUser/audit", ResourceType: "iam:AccessKey"},
			},
		},
	}
}
