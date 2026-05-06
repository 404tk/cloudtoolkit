package api

const CloudBillingBaseURL = "https://cloudbilling.googleapis.com"

// BillingAccount mirrors `google.cloud.billing.v1.BillingAccount`. The
// validation flow surfaces "open" billing accounts as the closest analogue
// to the alibaba/tencent "credit balance" — Google does not expose an
// account balance directly via API, so listing the active billing accounts
// is the management-plane signal CSPM detectors track.
type BillingAccount struct {
	Name                 string `json:"name"`
	DisplayName          string `json:"displayName"`
	Open                 bool   `json:"open"`
	MasterBillingAccount string `json:"masterBillingAccount"`
}

type ListBillingAccountsResponse struct {
	BillingAccounts []BillingAccount `json:"billingAccounts"`
	NextPageToken   string           `json:"nextPageToken"`
}
