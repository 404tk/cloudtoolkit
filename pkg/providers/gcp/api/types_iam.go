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

// ServiceAccountKey represents the resource returned by
// projects.serviceAccounts.keys.{list,create,get}. PrivateKeyData is only
// populated by `create` and is base64 of the credential JSON.
type ServiceAccountKey struct {
	Name            string `json:"name"`
	KeyAlgorithm    string `json:"keyAlgorithm,omitempty"`
	PrivateKeyType  string `json:"privateKeyType,omitempty"`
	PrivateKeyData  string `json:"privateKeyData,omitempty"`
	ValidAfterTime  string `json:"validAfterTime,omitempty"`
	ValidBeforeTime string `json:"validBeforeTime,omitempty"`
	KeyOrigin       string `json:"keyOrigin,omitempty"`
	KeyType         string `json:"keyType,omitempty"`
	Disabled        bool   `json:"disabled,omitempty"`
}

type ListServiceAccountKeysResponse struct {
	Keys []ServiceAccountKey `json:"keys"`
}

// CreateServiceAccountKeyRequest is the body of POST .../keys. CTK uses the
// default key type (TYPE_GOOGLE_CREDENTIALS_FILE) by leaving the field empty.
type CreateServiceAccountKeyRequest struct {
	PrivateKeyType string `json:"privateKeyType,omitempty"`
	KeyAlgorithm   string `json:"keyAlgorithm,omitempty"`
}
