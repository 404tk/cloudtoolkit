package replay

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleECS(req *http.Request, _ string, _ []byte) (*http.Response, error) {
	path := req.URL.Path
	method := strings.ToUpper(req.Method)
	if method != http.MethodGet || !strings.HasPrefix(path, "/v1/") || !strings.HasSuffix(path, "/cloudservers/detail") {
		return apiErrorResponse(req, http.StatusNotFound, "Ecs.0617",
			fmt.Sprintf("unsupported ecs path: %s %s", method, path)), nil
	}
	rest := strings.TrimPrefix(path, "/v1/")
	parts := strings.SplitN(rest, "/cloudservers/", 2)
	if len(parts) != 2 {
		return apiErrorResponse(req, http.StatusBadRequest, "Ecs.0001", "malformed cloudservers path"), nil
	}
	project, ok := findProjectByID(parts[0])
	if !ok {
		return apiErrorResponse(req, http.StatusForbidden, "Ecs.0617",
			fmt.Sprintf("project %s not visible to current user", parts[0])), nil
	}

	query := req.URL.Query()
	limit, _ := strconv.Atoi(strings.TrimSpace(query.Get("limit")))
	if limit <= 0 {
		limit = 100
	}
	offset, _ := strconv.Atoi(strings.TrimSpace(query.Get("offset")))
	if offset <= 0 {
		offset = 1
	}

	hosts := ecsHostsForRegion(project.Name)
	total := len(hosts)
	start := (offset - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	page := hosts[start:end]

	resp := api.ListECSServersDetailsResponse{Count: int32(total)}
	for _, host := range page {
		entry := api.ECSServerDetail{
			Status: host.Status,
			Name:   host.Name,
			Addresses: map[string][]api.ECSServerAddress{
				"vpc-replay": {
					{Addr: host.PrivateIP, OSEXTIPStype: "fixed"},
				},
			},
		}
		if host.PublicIP != "" {
			entry.Addresses["vpc-replay"] = append(entry.Addresses["vpc-replay"], api.ECSServerAddress{
				Addr: host.PublicIP, OSEXTIPStype: "floating",
			})
		}
		resp.Servers = append(resp.Servers, entry)
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}
