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

// SQLInstance is the typed Cloud SQL instance shape (instances.list).
type SQLInstance struct {
	Name             string                  `json:"name"`
	DatabaseVersion  string                  `json:"databaseVersion"`
	Region           string                  `json:"region"`
	State            string                  `json:"state"`
	IPAddresses      []SQLInstanceIPAddress  `json:"ipAddresses"`
	BackendType      string                  `json:"backendType"`
	InstanceType     string                  `json:"instanceType"`
	GceZone          string                  `json:"gceZone"`
	ConnectionName   string                  `json:"connectionName"`
	Settings         SQLInstanceSettings     `json:"settings"`
}

type SQLInstanceIPAddress struct {
	Type      string `json:"type"`
	IPAddress string `json:"ipAddress"`
}

type SQLInstanceSettings struct {
	Tier            string `json:"tier"`
	IPConfiguration struct {
		IPv4Enabled bool   `json:"ipv4Enabled"`
		PrivateNetwork string `json:"privateNetwork,omitempty"`
	} `json:"ipConfiguration"`
}

// SQLInstancesListResponse is the typed result of `instances.list`.
type SQLInstancesListResponse struct {
	Items         []SQLInstance `json:"items"`
	NextPageToken string        `json:"nextPageToken"`
	Kind          string        `json:"kind,omitempty"`
}
