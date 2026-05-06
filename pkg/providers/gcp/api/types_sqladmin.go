package api

// Cloud SQL Admin — user lifecycle endpoints used by rds-account-check.

const SQLAdminBaseURL = "https://sqladmin.googleapis.com"

type SQLUser struct {
	Name     string `json:"name"`
	Host     string `json:"host,omitempty"`
	Password string `json:"password,omitempty"`
	Project  string `json:"project,omitempty"`
	Instance string `json:"instance,omitempty"`
	Type     string `json:"type,omitempty"`
}

type SQLUsersListResponse struct {
	Items []SQLUser `json:"items"`
	Kind  string    `json:"kind,omitempty"`
}

type SQLOperation struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	OperationType string `json:"operationType,omitempty"`
}
