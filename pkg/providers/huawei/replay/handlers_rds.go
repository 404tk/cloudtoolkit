package replay

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleRDS(req *http.Request, _ string, _ []byte) (*http.Response, error) {
	path := req.URL.Path
	method := strings.ToUpper(req.Method)
	if method != http.MethodGet || !strings.HasPrefix(path, "/v3/") || !strings.HasSuffix(path, "/instances") {
		return apiErrorResponse(req, http.StatusNotFound, "DBS.200001",
			fmt.Sprintf("unsupported rds path: %s %s", method, path)), nil
	}
	rest := strings.TrimPrefix(path, "/v3/")
	parts := strings.SplitN(rest, "/instances", 2)
	if len(parts) < 1 || strings.TrimSpace(parts[0]) == "" {
		return apiErrorResponse(req, http.StatusBadRequest, "DBS.200002", "malformed instances path"), nil
	}
	project, ok := findProjectByID(parts[0])
	if !ok {
		return apiErrorResponse(req, http.StatusForbidden, "DBS.200611",
			fmt.Sprintf("project %s not visible to current user", parts[0])), nil
	}

	query := req.URL.Query()
	limit, _ := strconv.Atoi(strings.TrimSpace(query.Get("limit")))
	if limit <= 0 {
		limit = 100
	}
	offset, _ := strconv.Atoi(strings.TrimSpace(query.Get("offset")))

	instances := rdsInstancesForRegion(project.Name)
	total := int32(len(instances))
	start := offset
	if start < 0 {
		start = 0
	}
	if start > len(instances) {
		start = len(instances)
	}
	end := start + limit
	if end > len(instances) {
		end = len(instances)
	}
	page := instances[start:end]

	resp := api.ListRDSInstancesResponse{TotalCount: &total}
	for _, item := range page {
		entry := api.RDSInstance{
			ID:        item.ID,
			Region:    item.Region,
			Port:      item.Port,
			Datastore: &api.RDSDatastore{Type: item.Engine, Version: item.Version},
		}
		if item.PrivateIP != "" {
			entry.PrivateIPs = []string{item.PrivateIP}
		}
		if item.PublicIP != "" {
			entry.PublicIPs = []string{item.PublicIP}
		}
		resp.Instances = append(resp.Instances, entry)
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}
