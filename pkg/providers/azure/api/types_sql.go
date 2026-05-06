package api

const SQLAPIVersion = "2022-05-01-preview"

type SQLServerProperties struct {
	AdministratorLogin         string `json:"administratorLogin,omitempty"`
	AdministratorLoginPassword string `json:"administratorLoginPassword,omitempty"`
	Version                    string `json:"version,omitempty"`
	State                      string `json:"state,omitempty"`
	FullyQualifiedDomainName   string `json:"fullyQualifiedDomainName,omitempty"`
}

type SQLServerPatch struct {
	Properties SQLServerProperties `json:"properties"`
}

type SQLServer struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Location   string              `json:"location"`
	Properties SQLServerProperties `json:"properties"`
}
