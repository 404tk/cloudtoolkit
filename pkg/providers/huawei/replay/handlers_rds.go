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
	switch {
	case method == http.MethodPost && strings.HasPrefix(path, "/v3/") && strings.HasSuffix(path, "/db_user"):
		return t.handleRDSCreateAccount(req, path)
	case method == http.MethodDelete && strings.Contains(path, "/db_user/"):
		return t.handleRDSDeleteAccount(req, path)
	case method == http.MethodGet && strings.HasPrefix(path, "/v3/") && strings.HasSuffix(path, "/instances"):
		return t.handleRDSListInstances(req, path)
	}
	return apiErrorResponse(req, http.StatusNotFound, "DBS.200001",
		fmt.Sprintf("unsupported rds path: %s %s", method, path)), nil
}

func (t *transport) handleRDSListInstances(req *http.Request, path string) (*http.Response, error) {
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

// handleRDSCreateAccount accepts POST /v3/{project}/instances/{id}/db_user.
// We don't validate the project here because the body is what carries the
// useradd intent we're auditing.
func (t *transport) handleRDSCreateAccount(req *http.Request, path string) (*http.Response, error) {
	instanceID, ok := extractRDSInstanceID(path, "/db_user")
	if !ok {
		return apiErrorResponse(req, http.StatusBadRequest, "DBS.200002", "malformed db_user path"), nil
	}
	t.addHuaweiRDSAccount(instanceID, "ctkuser")
	return demoreplay.JSONResponse(req, http.StatusOK, api.CreateRDSDBUserResponse{Resp: "successful"}), nil
}

func (t *transport) handleRDSDeleteAccount(req *http.Request, path string) (*http.Response, error) {
	idx := strings.Index(path, "/db_user/")
	if idx < 0 {
		return apiErrorResponse(req, http.StatusBadRequest, "DBS.200002", "malformed db_user path"), nil
	}
	user := strings.TrimSpace(strings.TrimPrefix(path[idx:], "/db_user/"))
	instanceID, ok := extractRDSInstanceID(path[:idx]+"/db_user", "/db_user")
	if !ok || user == "" {
		return apiErrorResponse(req, http.StatusBadRequest, "DBS.200002", "malformed db_user path"), nil
	}
	if !t.removeHuaweiRDSAccount(instanceID, user) {
		return apiErrorResponse(req, http.StatusNotFound, "DBS.200013",
			fmt.Sprintf("user %s not found on %s", user, instanceID)), nil
	}
	return demoreplay.JSONResponse(req, http.StatusOK, api.DeleteRDSDBUserResponse{Resp: "successful"}), nil
}

// extractRDSInstanceID parses `<...>/instances/<id>/<suffix>` and returns the
// instance ID. Returns false if the segment can't be located.
func extractRDSInstanceID(path, suffix string) (string, bool) {
	rest := strings.TrimSuffix(path, suffix)
	idx := strings.LastIndex(rest, "/instances/")
	if idx < 0 {
		return "", false
	}
	id := strings.TrimSpace(rest[idx+len("/instances/"):])
	if id == "" {
		return "", false
	}
	return id, true
}
