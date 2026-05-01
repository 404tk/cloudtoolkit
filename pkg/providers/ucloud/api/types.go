package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type BaseResponse struct {
	Action  string  `json:"Action"`
	RetCode RetCode `json:"RetCode"`
	Message string  `json:"Message"`
}

type RetCode int

func (r *RetCode) UnmarshalJSON(data []byte) error {
	retCode, err := parseRetCode(data)
	if err != nil {
		return err
	}
	*r = RetCode(retCode)
	return nil
}

func parseRetCode(raw json.RawMessage) (int, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0, nil
	}

	var retCode int
	if err := json.Unmarshal(raw, &retCode); err == nil {
		return retCode, nil
	}

	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, fmt.Errorf("decode ucloud RetCode: %w", err)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	retCode, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("decode ucloud RetCode %q: %w", text, err)
	}
	return retCode, nil
}

type UserInfo struct {
	UserEmail string `json:"UserEmail"`
	UserID    int    `json:"UserId"`
	UserName  string `json:"UserName"`
}

type IAMUserSummary struct {
	CreatedAt   int64  `json:"CreatedAt"`
	DisplayName string `json:"DisplayName"`
	Email       string `json:"Email"`
	Status      string `json:"Status"`
	UserName    string `json:"UserName"`
}

type IAMPolicyInfo struct {
	PolicyName string `json:"PolicyName"`
	PolicyURN  string `json:"PolicyURN"`
}

type MemberProjectInfo struct {
	CharacterID string `json:"CharacterId"`
	ProjectID   string `json:"ProjectId"`
	ProjectName string `json:"ProjectName"`
}

type MemberInfo struct {
	ActivateFlag int                 `json:"ActivateFlag"`
	Created      int64               `json:"Created"`
	LastLogin    int64               `json:"LastLogin"`
	MemberEmail  string              `json:"MemberEmail"`
	MemberName   string              `json:"MemberName"`
	ProjectSet   []MemberProjectInfo `json:"ProjectSet"`
	State        string              `json:"State"`
}

type ProjectListInfo struct {
	IsDefault   bool   `json:"IsDefault"`
	ProjectID   string `json:"ProjectId"`
	ProjectName string `json:"ProjectName"`
}

type RegionInfo struct {
	Region string `json:"Region"`
}

type AccountInfo struct {
	Amount          string `json:"Amount"`
	AmountAvailable string `json:"AmountAvailable"`
}

type UHostIPSet struct {
	Default string `json:"Default"`
	IP      string `json:"IP"`
	IPMode  string `json:"IPMode"`
	Type    string `json:"Type"`
	Weight  int    `json:"Weight"`
}

type UHostSet struct {
	IPSet   []UHostIPSet `json:"IPSet"`
	Name    string       `json:"Name"`
	OsType  string       `json:"OsType"`
	State   string       `json:"State"`
	UHostID string       `json:"UHostId"`
}

type UFileBucketSet struct {
	BucketName string `json:"BucketName"`
	Region     string `json:"Region"`
	Type       string `json:"Type"`
}

// UpdateBucketResponse maps the JSON-RPC `UpdateBucket` action used to flip
// a UFile bucket's access type between private / public / limited (受限公开读).
type UpdateBucketResponse struct {
	BaseResponse
	BucketName string `json:"BucketName"`
	BucketID   string `json:"BucketId"`
}

type ValueSet struct {
	Data      string `json:"Data"`
	IsEnabled int    `json:"IsEnabled"`
}

type RecordInfo struct {
	Name     string     `json:"Name"`
	Type     string     `json:"Type"`
	ValueSet []ValueSet `json:"ValueSet"`
}

type ZoneInfo struct {
	DNSZoneID   string `json:"DNSZoneId"`
	DNSZoneName string `json:"DNSZoneName"`
}

type UDBInstanceSet struct {
	DBID         string `json:"DBId"`
	DBSubVersion string `json:"DBSubVersion"`
	DBTypeID     string `json:"DBTypeId"`
	Name         string `json:"Name"`
	Port         int    `json:"Port"`
	SubnetID     string `json:"SubnetId"`
	VirtualIP    string `json:"VirtualIP"`
	VPCID        string `json:"VPCId"`
}

type GetUserInfoResponse struct {
	BaseResponse
	DataSet []UserInfo `json:"DataSet"`
}

type GetProjectListResponse struct {
	BaseResponse
	ProjectSet []ProjectListInfo `json:"ProjectSet"`
}

type GetRegionResponse struct {
	BaseResponse
	Regions []RegionInfo `json:"Regions"`
}

type GetBalanceResponse struct {
	BaseResponse
	AccountInfo AccountInfo `json:"AccountInfo"`
}

type IAMListUsersResponse struct {
	BaseResponse
	TotalCount int              `json:"TotalCount"`
	Users      []IAMUserSummary `json:"Users"`
}

type IAMCreateUserResponse struct {
	BaseResponse
	APIAccess       bool   `json:"APIAccess"`
	AccessKeyID     string `json:"AccessKeyID"`
	AccessKeySecret string `json:"AccessKeySecret"`
	CompanyID       int64  `json:"CompanyId"`
	ConsoleAccess   bool   `json:"ConsoleAccess"`
	DisplayName     string `json:"DisplayName"`
	Password        string `json:"Password"`
	UserName        string `json:"UserName"`
}

type IAMDeleteUserResponse struct {
	BaseResponse
}

type IAMListPoliciesResponse struct {
	BaseResponse
	Policies   []IAMPolicyInfo `json:"Policies"`
	TotalCount int             `json:"TotalCount"`
}

type IAMAttachPoliciesToUserResponse struct {
	BaseResponse
}

// IAMDetachPoliciesFromUserResponse is the no-content reply from the
// `DetachPoliciesFromUser` action.
type IAMDetachPoliciesFromUserResponse struct {
	BaseResponse
}

// IAMListPoliciesForUserResponse maps the `ListPoliciesForUser` action that
// enumerates the policies bound to a sub user. Field names mirror the URN-
// based attachments documented for `AttachPoliciesToUser`.
type IAMListPoliciesForUserResponse struct {
	BaseResponse
	TotalCount int                      `json:"TotalCount"`
	Policies   []IAMUserAttachedPolicy  `json:"Policies"`
}

type IAMUserAttachedPolicy struct {
	PolicyURN  string `json:"PolicyURN"`
	PolicyName string `json:"PolicyName,omitempty"`
	Scope      string `json:"Scope,omitempty"`
	ProjectID  string `json:"ProjectID,omitempty"`
}

type IAMRemoveUserFromProjectResponse struct {
	BaseResponse
}

type DescribeMemberListResponse struct {
	BaseResponse
	MemberSet  []MemberInfo `json:"MemberSet"`
	TotalCount int          `json:"TotalCount"`
}

type DescribeUHostInstanceResponse struct {
	BaseResponse
	TotalCount int        `json:"TotalCount"`
	UHostSet   []UHostSet `json:"UHostSet"`
}

type DescribeBucketResponse struct {
	BaseResponse
	DataSet []UFileBucketSet `json:"DataSet"`
}

type DescribeUDNSZoneResponse struct {
	BaseResponse
	DNSZoneInfos []ZoneInfo `json:"DNSZoneInfos"`
	TotalCount   int        `json:"TotalCount"`
}

type DescribeUDNSRecordResponse struct {
	BaseResponse
	RecordInfos []RecordInfo `json:"RecordInfos"`
	TotalCount  int          `json:"TotalCount"`
}

type DescribeUDBInstanceResponse struct {
	BaseResponse
	DataSet    []UDBInstanceSet `json:"DataSet"`
	TotalCount int              `json:"TotalCount"`
}
