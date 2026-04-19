package api

const SubscriptionsAPIVersion = "2021-01-01"

type Subscription struct {
	SubscriptionID string `json:"subscriptionId"`
	DisplayName    string `json:"displayName"`
	State          string `json:"state"`
}

type ListSubscriptionsResponse struct {
	Value    []Subscription `json:"value"`
	NextLink string         `json:"nextLink"`
}
