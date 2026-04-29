package replay

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleOSS(req *http.Request) (*http.Response, error) {
	method := strings.ToUpper(req.Method)
	path := req.URL.Path
	switch {
	case method == http.MethodGet && path == "/v1/regions/cn-north-1/buckets":
		return t.handleListBuckets(req)
	case method == http.MethodHead && strings.HasPrefix(path, "/v1/regions/") && strings.Contains(path, "/buckets/"):
		return t.handleHeadBucket(req)
	}
	return apiErrorResponse(req, http.StatusNotFound, "NotFound",
		fmt.Sprintf("unsupported oss path: %s %s", method, path)), nil
}

func (t *transport) handleListBuckets(req *http.Request) (*http.Response, error) {
	resp := api.ListBucketsResponse{RequestID: "req-replay-oss-list"}
	for _, bucket := range demoBuckets {
		resp.Result.Buckets = append(resp.Result.Buckets, api.Bucket{Name: bucket.Name})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleHeadBucket(req *http.Request) (*http.Response, error) {
	rest := strings.TrimPrefix(req.URL.Path, "/v1/regions/")
	parts := strings.SplitN(rest, "/buckets/", 2)
	if len(parts) != 2 {
		return apiErrorResponse(req, http.StatusBadRequest, "InvalidPath",
			"malformed bucket head path"), nil
	}
	region := strings.TrimSpace(parts[0])
	bucket := strings.TrimSpace(parts[1])
	for _, item := range demoBuckets {
		if item.Name == bucket {
			if region == "cn-north-1" {
				resp := demoreplay.JSONResponse(req, http.StatusOK, struct {
					RequestID string `json:"requestId"`
				}{RequestID: "req-replay-oss-head"})
				return resp, nil
			}
			return apiErrorResponse(req, http.StatusNotFound, "NoSuchBucket",
				"bucket exists in a different region"), nil
		}
	}
	return apiErrorResponse(req, http.StatusNotFound, "NoSuchBucket",
		fmt.Sprintf("bucket %s not found", bucket)), nil
}
