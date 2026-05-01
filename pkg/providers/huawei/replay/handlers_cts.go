package replay

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleCTS(req *http.Request, region string) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "CTS.0001", "cts replay expects GET"), nil
	}

	projectID, ok := ctsProjectID(req.URL.Path)
	if !ok {
		return apiErrorResponse(req, http.StatusNotFound, "CTS.0001",
			fmt.Sprintf("unsupported cts path: %s", req.URL.Path)), nil
	}
	project, ok := findProjectByID(projectID)
	if !ok {
		return apiErrorResponse(req, http.StatusNotFound, "CTS.0007",
			fmt.Sprintf("project %s not found", projectID)), nil
	}
	if region != "" && project.Name != region {
		return apiErrorResponse(req, http.StatusNotFound, "CTS.0007",
			fmt.Sprintf("project %s does not belong to region %s", projectID, region)), nil
	}

	resp := api.ListTracesResponse{}
	for _, trace := range ctsTracesForRegion(project.Name) {
		resp.Traces = append(resp.Traces, api.Trace{
			TraceID:      trace.TraceID,
			TraceName:    trace.TraceName,
			TraceRating:  trace.TraceRating,
			TraceType:    trace.TraceType,
			Code:         trace.Code,
			APIService:   trace.ServiceType,
			OperationID:  trace.OperationID,
			ResourceID:   trace.ResourceID,
			ResourceName: trace.ResourceName,
			ResourceType: trace.ResourceType,
			SourceIP:     trace.SourceIP,
			Time:         trace.Time,
			User: api.TraceUser{
				AccessKeyID: trace.AccessKeyID,
				UserName:    demoUserName,
				Name:        demoUserName,
			},
		})
	}
	resp.MetaData.Count = len(resp.Traces)
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func ctsProjectID(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "v3" || parts[2] != "traces" {
		return "", false
	}
	projectID := strings.TrimSpace(parts[1])
	if projectID == "" {
		return "", false
	}
	return projectID, true
}
