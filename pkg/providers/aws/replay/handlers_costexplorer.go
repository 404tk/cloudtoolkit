package replay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleCostExplorer(req *http.Request, body []byte) (*http.Response, error) {
	target := strings.TrimSpace(req.Header.Get("X-Amz-Target"))
	switch target {
	case "AWSInsightsIndexService.GetCostAndUsage":
		return handleCEGetCostAndUsage(req, body)
	}
	return apiErrorResponse(req, http.StatusBadRequest, "InvalidAction",
		fmt.Sprintf("unsupported ce target: %s", target)), nil
}

func handleCEGetCostAndUsage(req *http.Request, body []byte) (*http.Response, error) {
	var input api.GetCostAndUsageInput
	if len(body) > 0 {
		if err := json.Unmarshal(body, &input); err != nil {
			return apiErrorResponse(req, http.StatusBadRequest, "ValidationError", err.Error()), nil
		}
	}
	out := api.GetCostAndUsageOutput{
		ResultsByTime: []api.CostResultByTime{{
			TimePeriod: input.TimePeriod,
			Total: map[string]api.CostMetric{
				"UnblendedCost": {Amount: "1284.5700", Unit: "USD"},
			},
			Estimated: true,
		}},
	}
	return demoreplay.JSONResponse(req, http.StatusOK, out), nil
}
