package replay

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleAsset(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			"asset service expects GET requests"), nil
	}
	if !strings.HasPrefix(req.URL.Path, "/v1/regions/") || !strings.HasSuffix(req.URL.Path, "/assets:describeAccountAmount") {
		return apiErrorResponse(req, http.StatusNotFound, "NotFound",
			fmt.Sprintf("unsupported asset path: %s", req.URL.Path)), nil
	}
	resp := api.DescribeAccountAmountResponse{RequestID: "req-replay-asset"}
	resp.Result.TotalAmount = "10240.00"
	resp.Result.AvailableAmount = "10240.00"
	resp.Result.FrozenAmount = "0.00"
	resp.Result.EnableWithdrawAmount = "10240.00"
	resp.Result.WithdrawingAmount = "0.00"
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}
