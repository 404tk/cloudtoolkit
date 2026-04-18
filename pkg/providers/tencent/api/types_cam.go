package api

import "context"

const (
	camVersion              = "2019-01-16"
	defaultCAMPolicyPage    = 1
	defaultCAMPolicyLimit   = 20
	defaultPolicyID         = 1
	defaultRoleSessionLimit = 10000
)

type ListUsersRequest struct{}

type ListUsersResponse struct {
	Response struct {
		Data      []SubAccountInfo `json:"Data"`
		RequestID string           `json:"RequestId"`
	} `json:"Response"`
}

type SubAccountInfo struct {
	Uin          *uint64 `json:"Uin"`
	Name         *string `json:"Name"`
	ConsoleLogin *uint64 `json:"ConsoleLogin"`
	CreateTime   *string `json:"CreateTime"`
}

func (c *Client) ListUsers(ctx context.Context) (ListUsersResponse, error) {
	var resp ListUsersResponse
	err := c.DoJSON(ctx, "cam", camVersion, "ListUsers", "", ListUsersRequest{}, &resp)
	return resp, err
}

type ListAttachedUserAllPoliciesRequest struct {
	TargetUin    *uint64 `json:"TargetUin,omitempty"`
	Rp           *uint64 `json:"Rp,omitempty"`
	Page         *uint64 `json:"Page,omitempty"`
	AttachType   *uint64 `json:"AttachType,omitempty"`
	StrategyType *uint64 `json:"StrategyType,omitempty"`
	Keyword      *string `json:"Keyword,omitempty"`
}

type ListAttachedUserAllPoliciesResponse struct {
	Response struct {
		PolicyList []AttachedUserPolicy `json:"PolicyList"`
		TotalNum   *uint64              `json:"TotalNum"`
		RequestID  string               `json:"RequestId"`
	} `json:"Response"`
}

type AttachedUserPolicy struct {
	PolicyID     *string `json:"PolicyId"`
	PolicyName   *string `json:"PolicyName"`
	StrategyType *string `json:"StrategyType"`
}

func (c *Client) ListAttachedUserAllPolicies(ctx context.Context, targetUin, page, rp, attachType uint64) (ListAttachedUserAllPoliciesResponse, error) {
	if page == 0 {
		page = defaultCAMPolicyPage
	}
	if rp == 0 {
		rp = defaultCAMPolicyLimit
	}
	var resp ListAttachedUserAllPoliciesResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"ListAttachedUserAllPolicies",
		"",
		ListAttachedUserAllPoliciesRequest{
			TargetUin:  uint64Ptr(targetUin),
			Rp:         uint64Ptr(rp),
			Page:       uint64Ptr(page),
			AttachType: uint64Ptr(attachType),
		},
		&resp,
	)
	return resp, err
}

type GetPolicyRequest struct {
	PolicyID *uint64 `json:"PolicyId,omitempty"`
}

type GetPolicyResponse struct {
	Response struct {
		PolicyDocument *string `json:"PolicyDocument"`
		RequestID      string  `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) GetPolicy(ctx context.Context, policyID uint64) (GetPolicyResponse, error) {
	var resp GetPolicyResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"GetPolicy",
		"",
		GetPolicyRequest{PolicyID: uint64Ptr(policyID)},
		&resp,
	)
	return resp, err
}

type AddUserRequest struct {
	Name              *string `json:"Name,omitempty"`
	ConsoleLogin      *uint64 `json:"ConsoleLogin,omitempty"`
	Password          *string `json:"Password,omitempty"`
	NeedResetPassword *uint64 `json:"NeedResetPassword,omitempty"`
}

type AddUserResponse struct {
	Response struct {
		Uin       *uint64 `json:"Uin"`
		Name      *string `json:"Name"`
		Password  *string `json:"Password"`
		RequestID string  `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) AddUser(ctx context.Context, name, password string) (AddUserResponse, error) {
	var resp AddUserResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"AddUser",
		"",
		AddUserRequest{
			Name:              stringPtr(name),
			ConsoleLogin:      uint64Ptr(1),
			Password:          stringPtr(password),
			NeedResetPassword: uint64Ptr(0),
		},
		&resp,
	)
	return resp, err
}

type AttachUserPolicyRequest struct {
	PolicyID  *uint64 `json:"PolicyId,omitempty"`
	AttachUin *uint64 `json:"AttachUin,omitempty"`
}

type AttachUserPolicyResponse struct {
	Response struct {
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) AttachUserPolicy(ctx context.Context, attachUin, policyID uint64) (AttachUserPolicyResponse, error) {
	if policyID == 0 {
		policyID = defaultPolicyID
	}
	var resp AttachUserPolicyResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"AttachUserPolicy",
		"",
		AttachUserPolicyRequest{
			PolicyID:  uint64Ptr(policyID),
			AttachUin: uint64Ptr(attachUin),
		},
		&resp,
	)
	return resp, err
}

type DetachUserPolicyRequest struct {
	PolicyID  *uint64 `json:"PolicyId,omitempty"`
	DetachUin *uint64 `json:"DetachUin,omitempty"`
}

type DetachUserPolicyResponse struct {
	Response struct {
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) DetachUserPolicy(ctx context.Context, detachUin, policyID uint64) (DetachUserPolicyResponse, error) {
	if policyID == 0 {
		policyID = defaultPolicyID
	}
	var resp DetachUserPolicyResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"DetachUserPolicy",
		"",
		DetachUserPolicyRequest{
			PolicyID:  uint64Ptr(policyID),
			DetachUin: uint64Ptr(detachUin),
		},
		&resp,
	)
	return resp, err
}

type GetUserRequest struct {
	Name *string `json:"Name,omitempty"`
}

type GetUserResponse struct {
	Response struct {
		Uin       *uint64 `json:"Uin"`
		Name      *string `json:"Name"`
		RequestID string  `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) GetUser(ctx context.Context, name string) (GetUserResponse, error) {
	var resp GetUserResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"GetUser",
		"",
		GetUserRequest{Name: stringPtr(name)},
		&resp,
	)
	return resp, err
}

type GetUserAppIDRequest struct{}

type GetUserAppIDResponse struct {
	Response struct {
		OwnerUin  *string `json:"OwnerUin"`
		RequestID string  `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) GetUserAppID(ctx context.Context) (GetUserAppIDResponse, error) {
	var resp GetUserAppIDResponse
	err := c.DoJSON(ctx, "cam", camVersion, "GetUserAppId", "", GetUserAppIDRequest{}, &resp)
	return resp, err
}

type DeleteUserRequest struct {
	Name  *string `json:"Name,omitempty"`
	Force *uint64 `json:"Force,omitempty"`
}

type DeleteUserResponse struct {
	Response struct {
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) DeleteUser(ctx context.Context, name string, force uint64) (DeleteUserResponse, error) {
	var resp DeleteUserResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"DeleteUser",
		"",
		DeleteUserRequest{
			Name:  stringPtr(name),
			Force: uint64Ptr(force),
		},
		&resp,
	)
	return resp, err
}

type CreateRoleRequest struct {
	RoleName        *string `json:"RoleName,omitempty"`
	PolicyDocument  *string `json:"PolicyDocument,omitempty"`
	ConsoleLogin    *uint64 `json:"ConsoleLogin,omitempty"`
	SessionDuration *uint64 `json:"SessionDuration,omitempty"`
}

type CreateRoleResponse struct {
	Response struct {
		RoleID    *string `json:"RoleId"`
		RequestID string  `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) CreateRole(ctx context.Context, roleName, policyDocument string, consoleLogin, sessionDuration uint64) (CreateRoleResponse, error) {
	if sessionDuration == 0 {
		sessionDuration = defaultRoleSessionLimit
	}
	var resp CreateRoleResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"CreateRole",
		"",
		CreateRoleRequest{
			RoleName:        stringPtr(roleName),
			PolicyDocument:  stringPtr(policyDocument),
			ConsoleLogin:    uint64Ptr(consoleLogin),
			SessionDuration: uint64Ptr(sessionDuration),
		},
		&resp,
	)
	return resp, err
}

type AttachRolePolicyRequest struct {
	PolicyID       *uint64 `json:"PolicyId,omitempty"`
	AttachRoleName *string `json:"AttachRoleName,omitempty"`
}

type AttachRolePolicyResponse struct {
	Response struct {
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) AttachRolePolicy(ctx context.Context, roleName string, policyID uint64) (AttachRolePolicyResponse, error) {
	if policyID == 0 {
		policyID = defaultPolicyID
	}
	var resp AttachRolePolicyResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"AttachRolePolicy",
		"",
		AttachRolePolicyRequest{
			PolicyID:       uint64Ptr(policyID),
			AttachRoleName: stringPtr(roleName),
		},
		&resp,
	)
	return resp, err
}

type DetachRolePolicyRequest struct {
	PolicyID       *uint64 `json:"PolicyId,omitempty"`
	DetachRoleName *string `json:"DetachRoleName,omitempty"`
}

type DetachRolePolicyResponse struct {
	Response struct {
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) DetachRolePolicy(ctx context.Context, roleName string, policyID uint64) (DetachRolePolicyResponse, error) {
	if policyID == 0 {
		policyID = defaultPolicyID
	}
	var resp DetachRolePolicyResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"DetachRolePolicy",
		"",
		DetachRolePolicyRequest{
			PolicyID:       uint64Ptr(policyID),
			DetachRoleName: stringPtr(roleName),
		},
		&resp,
	)
	return resp, err
}

type DeleteRoleRequest struct {
	RoleName *string `json:"RoleName,omitempty"`
}

type DeleteRoleResponse struct {
	Response struct {
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) DeleteRole(ctx context.Context, roleName string) (DeleteRoleResponse, error) {
	var resp DeleteRoleResponse
	err := c.DoJSON(
		ctx,
		"cam",
		camVersion,
		"DeleteRole",
		"",
		DeleteRoleRequest{RoleName: stringPtr(roleName)},
		&resp,
	)
	return resp, err
}

func stringPtr(v string) *string {
	return &v
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}
