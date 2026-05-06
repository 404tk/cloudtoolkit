package replay

import (
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleDomainService serves the cloudlist `domain` asset endpoints used by the
// JDCloud DNS driver:
//   - GET /v2/regions/<region>/domain                                (list zones)
//   - GET /v2/regions/<region>/domain/<id>/ResourceRecord            (list RRs)
//
// Both endpoints honour pageNumber / pageSize via the shared paginationParams
// + windowSlice helpers so the replay surface exercises the driver's pagination
// loop instead of always returning the full fixture in one shot.
func (t *transport) handleDomainService(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			"unsupported domainservice method: "+req.Method), nil
	}
	path := strings.TrimRight(req.URL.Path, "/")
	switch {
	case strings.HasSuffix(path, "/ResourceRecord") && strings.Contains(path, "/domain/"):
		all := demoDomainRecords(extractDomainID(path))
		page, size := paginationParams(req)
		w := windowSlice(len(all), page, size)
		resp := api.DescribeResourceRecordResponse{RequestID: "req-replay-domain-resource-record"}
		resp.Result.DataList = all[w.start:w.end]
		resp.Result.CurrentCount = len(resp.Result.DataList)
		resp.Result.TotalCount = len(all)
		resp.Result.TotalPage = pageCount(len(all), size)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case strings.HasSuffix(path, "/domain"):
		all := demoDomains()
		page, size := paginationParams(req)
		w := windowSlice(len(all), page, size)
		resp := api.DescribeDomainsResponse{RequestID: "req-replay-domain-describe"}
		resp.Result.DataList = all[w.start:w.end]
		resp.Result.CurrentCount = len(resp.Result.DataList)
		resp.Result.TotalCount = len(all)
		resp.Result.TotalPage = pageCount(len(all), size)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "InvalidPath",
		"unsupported domainservice path: "+req.URL.Path), nil
}

// extractDomainID parses `/v2/regions/<region>/domain/<id>/ResourceRecord`.
func extractDomainID(path string) string {
	idx := strings.Index(path, "/domain/")
	if idx < 0 {
		return ""
	}
	rest := path[idx+len("/domain/"):]
	end := strings.Index(rest, "/")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

func pageCount(total, size int) int {
	if size <= 0 || total <= 0 {
		return 0
	}
	return (total + size - 1) / size
}

func demoDomains() []api.DomainInfo {
	return []api.DomainInfo{
		{
			ID:              1001,
			DomainName:      "ctk-demo-public.example.com",
			CreateTime:      1745020800000,
			ExpirationDate:  1776556800000,
			PackName:        "Enterprise",
			ResolvingStatus: "2",
			Creator:         "ctk-demo-admin",
			JcloudNs:        true,
		},
		{
			ID:              1002,
			DomainName:      "ctk-demo-internal.example",
			CreateTime:      1744934400000,
			ExpirationDate:  1776470400000,
			PackName:        "Free",
			ResolvingStatus: "2",
			Creator:         "ctk-demo-admin",
			JcloudNs:        true,
		},
	}
}

func demoDomainRecords(domainID string) []api.DomainResourceRecord {
	switch domainID {
	case "1001":
		return []api.DomainResourceRecord{
			{ID: 11, HostRecord: "@", Type: "A", HostValue: "198.51.100.10", TTL: 300, ResolvingStatus: "2"},
			{ID: 12, HostRecord: "www", Type: "CNAME", HostValue: "ctk-demo-public.example.com", TTL: 60, ResolvingStatus: "2"},
			{ID: 13, HostRecord: "@", Type: "MX", HostValue: "10 mx1.ctk-demo-public.example.com", TTL: 300, MxPriority: 10, ResolvingStatus: "4"},
		}
	case "1002":
		return []api.DomainResourceRecord{
			{ID: 21, HostRecord: "db", Type: "A", HostValue: "10.0.0.10", TTL: 60, ResolvingStatus: "2"},
		}
	}
	return nil
}
