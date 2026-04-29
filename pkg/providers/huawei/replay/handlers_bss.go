package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleBSS(req *http.Request, _ []byte) (*http.Response, error) {
	if req.URL.Path != "/v2/accounts/customer-accounts/balances" {
		return apiErrorResponse(req, http.StatusNotFound, "CBC.0001",
			"unsupported bss path: "+req.URL.Path), nil
	}
	resp := api.ShowCustomerAccountBalancesResponse{
		AccountBalances: []api.AccountBalance{
			{AccountType: 1, Amount: "1024.88"},
			{AccountType: 2, Amount: "0.00"},
		},
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}
