package api

type ListRegionsResponse struct {
	Regions []Region `json:"regions"`
}

type Region struct {
	ID string `json:"id"`
}

type ShowPermanentAccessKeyResponse struct {
	Credential struct {
		UserID string `json:"user_id"`
	} `json:"credential"`
	ErrorMsg string `json:"error_msg"`
}

type ShowUserResponse struct {
	User struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		DomainID string `json:"domain_id"`
	} `json:"user"`
}

type ListUsersResponse struct {
	Users []IAMUser `json:"users"`
}

type IAMUser struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type ListUsersV5Response struct {
	Users []IAMUserV5 `json:"users"`
}

type IAMUserV5 struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Enabled  bool   `json:"enabled"`
}

type CreateUserRequest struct {
	User CreateUserOption `json:"user"`
}

type CreateUserOption struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
	DomainID string `json:"domain_id,omitempty"`
}

type CreateUserResponse struct {
	User struct {
		ID       string `json:"id"`
		DomainID string `json:"domain_id"`
	} `json:"user"`
}

type ListGroupsResponse struct {
	Groups []IAMGroup `json:"groups"`
}

type IAMGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListAuthDomainsResponse struct {
	Domains []IAMDomain `json:"domains"`
}

type IAMDomain struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListProjectsResponse struct {
	Projects []IAMProject `json:"projects"`
}

type IAMProject struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DomainID string `json:"domain_id"`
	Enabled  bool   `json:"enabled"`
}
