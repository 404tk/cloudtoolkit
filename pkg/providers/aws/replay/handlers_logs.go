package replay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleLogs(req *http.Request, body []byte) (*http.Response, error) {
	target := strings.TrimSpace(req.Header.Get("X-Amz-Target"))
	switch target {
	case "Logs_20140328.DescribeLogGroups":
		return handleLogsDescribeLogGroups(req, body)
	}
	return apiErrorResponse(req, http.StatusBadRequest, "InvalidAction",
		fmt.Sprintf("unsupported logs target: %s", target)), nil
}

func handleLogsDescribeLogGroups(req *http.Request, body []byte) (*http.Response, error) {
	if len(body) > 0 {
		var input api.DescribeLogGroupsInput
		if err := json.Unmarshal(body, &input); err != nil {
			return apiErrorResponse(req, http.StatusBadRequest, "ValidationError", err.Error()), nil
		}
	}
	region := regionFromHost(req.URL.Hostname())
	groups := logGroupsForRegion(region)
	out := api.DescribeLogGroupsOutput{}
	for _, g := range groups {
		out.LogGroups = append(out.LogGroups, api.LogGroup{
			LogGroupName:    g.Name,
			CreationTime:    g.CreationTime,
			RetentionInDays: g.RetentionInDays,
			StoredBytes:     g.StoredBytes,
			Arn:             g.Arn,
		})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, out), nil
}
