package replay

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleVM(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			"VM service expects GET requests"), nil
	}
	path := req.URL.Path
	if !strings.HasPrefix(path, "/v1/regions/") || !strings.HasSuffix(path, "/instances") {
		return apiErrorResponse(req, http.StatusNotFound, "NotFound",
			fmt.Sprintf("unsupported vm path: %s", path)), nil
	}
	region := strings.TrimSuffix(strings.TrimPrefix(path, "/v1/regions/"), "/instances")
	region = strings.TrimSpace(region)
	pageNumber, pageSize := paginationParams(req)
	all := vmsForRegion(region)
	page := windowSlice(len(all), pageNumber, pageSize)

	resp := api.DescribeInstancesResponse{RequestID: "req-replay-vm-list"}
	resp.Result.TotalCount = len(all)
	for _, inst := range all[page.start:page.end] {
		resp.Result.Instances = append(resp.Result.Instances, api.Instance{
			InstanceID:       inst.InstanceID,
			Hostname:         inst.Hostname,
			Status:           inst.Status,
			OSType:           inst.OSType,
			PrivateIPAddress: inst.PrivateIPAddress,
			ElasticIPAddress: inst.ElasticIPAddress,
		})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleLAVM(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			"LAVM service expects GET requests"), nil
	}
	path := req.URL.Path
	if !strings.HasPrefix(path, "/v1/regions/") || !strings.HasSuffix(path, "/instances") {
		return apiErrorResponse(req, http.StatusNotFound, "NotFound",
			fmt.Sprintf("unsupported lavm path: %s", path)), nil
	}
	region := strings.TrimSuffix(strings.TrimPrefix(path, "/v1/regions/"), "/instances")
	region = strings.TrimSpace(region)
	pageNumber, pageSize := paginationParams(req)
	all := lavmForRegion(region)
	page := windowSlice(len(all), pageNumber, pageSize)

	resp := api.DescribeLAVMInstancesResponse{RequestID: "req-replay-lavm-list"}
	resp.Result.TotalCount = len(all)
	for _, inst := range all[page.start:page.end] {
		resp.Result.Instances = append(resp.Result.Instances, api.LAVMInstance{
			InstanceID:       inst.InstanceID,
			InstanceName:     inst.InstanceName,
			Status:           inst.Status,
			RegionID:         inst.Region,
			PublicIPAddress:  inst.PublicIP,
			PrivateIPAddress: inst.PrivateIP,
			ImageID:          inst.ImageID,
			BusinessStatus:   inst.BusinessStatus,
		})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

type sliceWindow struct {
	start int
	end   int
}

func windowSlice(total, pageNumber, pageSize int) sliceWindow {
	if pageNumber <= 0 {
		pageNumber = 1
	}
	if pageSize <= 0 {
		pageSize = total
	}
	start := (pageNumber - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return sliceWindow{start: start, end: end}
}

func paginationParams(req *http.Request) (int, int) {
	query := req.URL.Query()
	page, _ := strconv.Atoi(strings.TrimSpace(query.Get("pageNumber")))
	if page <= 0 {
		page = 1
	}
	size, _ := strconv.Atoi(strings.TrimSpace(query.Get("pageSize")))
	if size <= 0 {
		size = 100
	}
	return page, size
}
