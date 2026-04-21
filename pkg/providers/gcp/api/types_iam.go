package api

const IAMBaseURL = "https://iam.googleapis.com"

type ServiceAccount struct {
	Name           string `json:"name"`
	ProjectID      string `json:"projectId"`
	UniqueID       string `json:"uniqueId"`
	Email          string `json:"email"`
	DisplayName    string `json:"displayName"`
	OAuth2ClientID string `json:"oauth2ClientId"`
	Disabled       bool   `json:"disabled"`
}

type ListServiceAccountsResponse struct {
	Accounts      []ServiceAccount `json:"accounts"`
	NextPageToken string           `json:"nextPageToken"`
}
