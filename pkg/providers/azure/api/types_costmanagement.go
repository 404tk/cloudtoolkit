package api

const CostManagementAPIVersion = "2023-11-01"

// CostManagementQueryRequest is the body of the Cost Management `query` POST.
// The Azure API supports a wide range of grouping/filter options; we keep the
// minimum that yields the current-month total for an account.
type CostManagementQueryRequest struct {
	Type        string                       `json:"type"`
	Timeframe   string                       `json:"timeframe"`
	Dataset     CostManagementQueryDataset   `json:"dataset"`
	TimePeriod  *CostManagementQueryTimePeriod `json:"timePeriod,omitempty"`
}

type CostManagementQueryTimePeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type CostManagementQueryDataset struct {
	Granularity string                                  `json:"granularity"`
	Aggregation map[string]CostManagementAggregation    `json:"aggregation"`
}

type CostManagementAggregation struct {
	Name     string `json:"name"`
	Function string `json:"function"`
}

// CostManagementQueryResponse mirrors the shape returned by `Microsoft.CostManagement
// /query`. Rows are interleaved Cost+Currency values with column descriptors.
type CostManagementQueryResponse struct {
	ID         string                          `json:"id"`
	Name       string                          `json:"name"`
	Type       string                          `json:"type"`
	Properties CostManagementQueryProperties   `json:"properties"`
}

type CostManagementQueryProperties struct {
	NextLink string                          `json:"nextLink"`
	Columns  []CostManagementColumn          `json:"columns"`
	Rows     [][]any                         `json:"rows"`
}

type CostManagementColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
