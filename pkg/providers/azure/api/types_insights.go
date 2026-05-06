package api

// Microsoft.Insights Activity Log REST surface used by event-check.

const InsightsAPIVersion = "2015-04-01"

// ActivityLogEvent maps a single record from the
// `Microsoft.Insights/eventtypes/management/values` listing. Only fields
// useful for the validation flow are projected.
type ActivityLogEvent struct {
	EventDataID string `json:"eventDataId"`
	OperationName struct {
		Value          string `json:"value"`
		LocalizedValue string `json:"localizedValue"`
	} `json:"operationName"`
	EventTimestamp string `json:"eventTimestamp"`
	Caller         string `json:"caller"`
	HTTPRequest    struct {
		ClientIPAddress string `json:"clientIpAddress"`
	} `json:"httpRequest"`
	ResourceID    string `json:"resourceId"`
	Status        struct {
		Value          string `json:"value"`
		LocalizedValue string `json:"localizedValue"`
	} `json:"status"`
	SubmissionTimestamp string `json:"submissionTimestamp"`
	Authorization       struct {
		Action string `json:"action"`
		Scope  string `json:"scope"`
	} `json:"authorization"`
	ResourceType struct {
		Value          string `json:"value"`
		LocalizedValue string `json:"localizedValue"`
	} `json:"resourceType"`
}

type ActivityLogResponse struct {
	Value    []ActivityLogEvent `json:"value"`
	NextLink string             `json:"nextLink"`
}
