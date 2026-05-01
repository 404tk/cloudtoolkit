package replay

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/oss"
	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/sls"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type invocationResult struct {
	Output string
}

type transport struct {
	mu          sync.Mutex
	invocations map[string]invocationResult
	userPolicy  map[string][]ramPolicyFixture
	bucketACL   map[string]string
}

func newTransport() *transport {
	return &transport{
		invocations: make(map[string]invocationResult),
		userPolicy:  seedUserPolicies(),
		bucketACL:   seedBucketACLs(),
	}
}

// seedUserPolicies copies the static policy attachments out of demoRAMUsers so
// the transport can mutate them per request without altering the fixture.
func seedUserPolicies() map[string][]ramPolicyFixture {
	out := make(map[string][]ramPolicyFixture, len(demoRAMUsers))
	for _, user := range demoRAMUsers {
		policies := make([]ramPolicyFixture, len(user.AttachedPolicy))
		copy(policies, user.AttachedPolicy)
		out[user.UserName] = policies
	}
	return out
}

// seedBucketACLs gives every demo bucket a starting "private" canned ACL so
// audit/expose/audit/unexpose cycles surface deterministic state.
func seedBucketACLs() map[string]string {
	out := make(map[string]string, len(demoBuckets))
	for _, bucket := range demoBuckets {
		out[bucket.Name] = "private"
	}
	return out
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case isOSSHost(req):
		return t.handleOSS(req)
	case isSLSHost(req):
		return t.handleSLS(req)
	default:
		return t.handleRPC(req)
	}
}

func (t *transport) handleRPC(req *http.Request) (*http.Response, error) {
	switch verifyRPCAuth(req) {
	case demoreplay.AuthInvalidAccessKey:
		return rpcErrorResponse(req, http.StatusForbidden, "InvalidAccessKeyId.NotFound", "Specified access key is not found."), nil
	case demoreplay.AuthInvalidSignature:
		return rpcErrorResponse(req, http.StatusForbidden, "SignatureDoesNotMatch", "Specified signature is not matched with our calculation."), nil
	}

	host := strings.ToLower(req.URL.Hostname())
	action := strings.TrimSpace(req.URL.Query().Get("Action"))
	switch rpcProductFromHost(host) {
	case "sts":
		if action == "GetCallerIdentity" {
			return demoreplay.JSONResponse(req, http.StatusOK, api.GetCallerIdentityResponse{
				IdentityType: "RAMUser",
				AccountID:    "235000000000000001",
				RequestID:    "req-sts-caller",
				PrincipalID:  "235000000000000001",
				UserID:       "235000000000000001",
				Arn:          demoCallerArn(),
			}), nil
		}
	case "bssopenapi":
		if action == "QueryAccountBalance" {
			return demoreplay.JSONResponse(req, http.StatusOK, api.QueryAccountBalanceResponse{
				Code:      "Success",
				Message:   "success",
				RequestID: "req-bss-balance",
				Success:   true,
				Data: api.AccountBalanceData{
					AvailableCashAmount: demoBalanceAmount(),
				},
			}), nil
		}
	case "ecs":
		return t.handleECS(req, action)
	case "ram":
		return t.handleRAM(req, action)
	case "rds":
		return t.handleRDS(req, action)
	case "sas":
		return t.handleSAS(req, action)
	case "alidns":
		return t.handleDNS(req, action)
	case "dysmsapi":
		return t.handleSMS(req, action)
	case "location":
		return t.handleLocation(req, action)
	}

	return rpcErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
}

func (t *transport) handleLocation(req *http.Request, action string) (*http.Response, error) {
	if action != "DescribeEndpoints" {
		return rpcErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported Location replay action: %s", action)), nil
	}

	serviceCode := strings.TrimSpace(req.URL.Query().Get("ServiceCode"))
	region := strings.TrimSpace(req.URL.Query().Get("Id"))
	if region == "" {
		region = api.DefaultRegion
	}
	host := ""
	switch serviceCode {
	case "ecs":
		host = replayECSEndpoint(region)
	}
	return demoreplay.JSONResponse(req, http.StatusOK, map[string]any{
		"RequestId": "req-location-endpoints",
		"Success":   true,
		"Endpoints": map[string]any{
			"Endpoint": []map[string]string{
				{"Endpoint": host},
			},
		},
	}), nil
}

func (t *transport) handleECS(req *http.Request, action string) (*http.Response, error) {
	query := req.URL.Query()
	switch action {
	case "DescribeRegions":
		regions := make([]api.ECSRegion, 0, len(demoRegions))
		for _, region := range demoRegions {
			regions = append(regions, api.ECSRegion{RegionID: region})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.DescribeECSRegionsResponse{
			RequestID: "req-ecs-regions",
			Regions:   api.ECSRegionList{Region: regions},
		}), nil
	case "DescribeInstances":
		region := strings.TrimSpace(query.Get("RegionId"))
		pageNumber := demoreplay.ParseInt(query.Get("PageNumber"), 1)
		pageSize := demoreplay.ParseInt(query.Get("PageSize"), 100)
		hosts := hostsForRegion(region)
		window := demoreplay.PageWindow(len(hosts), pageNumber, pageSize)
		items := make([]api.ECSInstance, 0, window.End-window.Start)
		for _, host := range hosts[window.Start:window.End] {
			instance := api.ECSInstance{
				HostName:   host.HostName,
				InstanceID: host.ID,
				OSType:     host.OSType,
				PublicIP:   api.ECSPublicIPList{IPAddress: demoreplay.NonEmptyStrings(host.PublicIPv4)},
				NetworkInterfaces: api.ECSNetworkInterfaces{
					NetworkInterface: []api.ECSNetworkInterface{
						{
							PrimaryIPAddress: host.PrivateIpv4,
							PrivateIPSets: api.ECSPrivateIPSets{
								PrivateIPSet: []api.ECSPrivateIPSet{
									{PrivateIPAddress: host.PrivateIpv4},
								},
							},
						},
					},
				},
			}
			items = append(items, instance)
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.DescribeECSInstancesResponse{
			PageSize:   pageSize,
			PageNumber: pageNumber,
			RequestID:  "req-ecs-instances",
			TotalCount: len(hosts),
			Instances:  api.ECSInstanceList{Instance: items},
		}), nil
	case "RunCommand":
		instanceID := strings.TrimSpace(demoreplay.FirstNonEmpty(query.Get("InstanceId.1"), query.Get("InstanceId")))
		command := strings.TrimSpace(query.Get("CommandContent"))
		if strings.EqualFold(query.Get("ContentEncoding"), "Base64") {
			if decoded, err := base64.StdEncoding.DecodeString(command); err == nil {
				command = string(decoded)
			}
		}
		commandID := buildCommandID(instanceID, command)
		t.mu.Lock()
		t.invocations[commandID] = invocationResult{
			Output: shellOutput(instanceID, command),
		}
		t.mu.Unlock()
		return demoreplay.JSONResponse(req, http.StatusOK, api.RunECSCommandResponse{
			RequestID: "req-ecs-run-command",
			CommandID: commandID,
			InvokeID:  "invoke-" + commandID,
		}), nil
	case "DescribeInvocationResults":
		commandID := strings.TrimSpace(query.Get("CommandId"))
		t.mu.Lock()
		result, ok := t.invocations[commandID]
		t.mu.Unlock()
		if !ok {
			return rpcErrorResponse(req, http.StatusNotFound, "InvalidCommandId.NotFound", "Specified command ID does not exist."), nil
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.DescribeECSInvocationResultsResponse{
			RequestID: "req-ecs-invocation",
			Invocation: api.ECSInvocation{
				CommandID: commandID,
				InvokeID:  "invoke-" + commandID,
				InvocationResults: api.ECSInvocationResults{
					InvocationResult: []api.ECSInvocationResult{
						{
							InvokeRecordStatus: "Finished",
							Output:             result.Output,
						},
					},
				},
			},
		}), nil
	}
	return rpcErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported ECS replay action: %s", action)), nil
}

func (t *transport) handleRAM(req *http.Request, action string) (*http.Response, error) {
	query := req.URL.Query()
	switch action {
	case "ListUsers":
		maxItems := demoreplay.ParseInt(query.Get("MaxItems"), 100)
		offset := markerOffset(query.Get("Marker"))
		window := demoreplay.OffsetWindow(len(demoRAMUsers), offset, maxItems)
		items := make([]api.RAMUser, 0, window.End-window.Start)
		for _, user := range demoRAMUsers[window.Start:window.End] {
			items = append(items, api.RAMUser{
				UserName:   user.UserName,
				UserID:     user.UserID,
				CreateDate: user.CreateDate,
			})
		}
		resp := api.ListRAMUsersResponse{
			RequestID:   "req-ram-list-users",
			IsTruncated: window.End < len(demoRAMUsers),
			Users:       api.RAMUserList{User: items},
		}
		if resp.IsTruncated {
			resp.Marker = strconv.Itoa(window.End)
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "GetLoginProfile":
		user, ok := findRAMUser(query.Get("UserName"))
		if !ok || !user.HasLogin {
			return rpcErrorResponse(req, http.StatusNotFound, "EntityNotExist.LoginProfile", "Login policy not exists"), nil
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.GetRAMLoginProfileResponse{
			RequestID: "req-ram-login-profile",
			LoginProfile: api.RAMLoginProfile{
				UserName:              user.UserName,
				CreateDate:            user.CreateDate,
				PasswordResetRequired: false,
				MFABindRequired:       false,
			},
		}), nil
	case "GetUser":
		user, ok := findRAMUser(query.Get("UserName"))
		if !ok {
			return rpcErrorResponse(req, http.StatusNotFound, "EntityNotExist.User", "The specified RAM user does not exist."), nil
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.GetRAMUserResponse{
			RequestID: "req-ram-get-user",
			User: api.RAMUser{
				UserName:      user.UserName,
				UserID:        user.UserID,
				CreateDate:    user.CreateDate,
				LastLoginDate: user.LastLoginDate,
			},
		}), nil
	case "ListPoliciesForUser":
		userName := strings.TrimSpace(query.Get("UserName"))
		if _, ok := findRAMUser(userName); !ok {
			return rpcErrorResponse(req, http.StatusNotFound, "EntityNotExist.User", "The specified RAM user does not exist."), nil
		}
		t.mu.Lock()
		attached := append([]ramPolicyFixture(nil), t.userPolicy[userName]...)
		t.mu.Unlock()
		policies := make([]api.RAMPolicy, 0, len(attached))
		for _, policy := range attached {
			policies = append(policies, api.RAMPolicy{
				PolicyName: policy.Name,
				PolicyType: policy.Type,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.ListRAMPoliciesForUserResponse{
			RequestID: "req-ram-list-policies",
			Policies:  api.RAMPolicyList{Policy: policies},
		}), nil
	case "GetPolicy":
		policyName := strings.TrimSpace(query.Get("PolicyName"))
		for _, user := range demoRAMUsers {
			for _, policy := range user.AttachedPolicy {
				if policy.Name == policyName {
					return demoreplay.JSONResponse(req, http.StatusOK, api.GetRAMPolicyResponse{
						RequestID: "req-ram-get-policy",
						Policy: api.RAMPolicy{
							PolicyName: policy.Name,
							PolicyType: policy.Type,
						},
						DefaultPolicyVersion: api.RAMDefaultPolicyVersion{
							IsDefaultVersion: true,
							VersionID:        "v1",
							PolicyDocument:   policy.Document,
						},
					}), nil
				}
			}
		}
		return rpcErrorResponse(req, http.StatusNotFound, "EntityNotExist.Policy", "The specified policy does not exist."), nil
	case "GetAccountAlias":
		return demoreplay.JSONResponse(req, http.StatusOK, api.GetRAMAccountAliasResponse{
			RequestID:    "req-ram-account-alias",
			AccountAlias: demoAccountAlias(),
		}), nil
	case "CreateUser":
		return demoreplay.JSONResponse(req, http.StatusOK, api.CreateRAMUserResponse{
			RequestID: "req-ram-create-user",
			User: api.RAMUser{
				UserName:   query.Get("UserName"),
				UserID:     "235000000000009999",
				CreateDate: "2026-04-20T10:00:00+08:00",
			},
		}), nil
	case "CreateLoginProfile":
		return demoreplay.JSONResponse(req, http.StatusOK, api.CreateRAMLoginProfileResponse{
			RequestID: "req-ram-create-login-profile",
			LoginProfile: api.RAMLoginProfile{
				UserName:              query.Get("UserName"),
				CreateDate:            "2026-04-20T10:00:00+08:00",
				PasswordResetRequired: false,
				MFABindRequired:       false,
			},
		}), nil
	case "AttachPolicyToUser":
		userName := strings.TrimSpace(query.Get("UserName"))
		policyName := strings.TrimSpace(query.Get("PolicyName"))
		policyType := strings.TrimSpace(query.Get("PolicyType"))
		if _, ok := findRAMUser(userName); !ok {
			return rpcErrorResponse(req, http.StatusNotFound, "EntityNotExist.User", "The specified RAM user does not exist."), nil
		}
		t.mu.Lock()
		policies := t.userPolicy[userName]
		for _, p := range policies {
			if p.Name == policyName && p.Type == policyType {
				t.mu.Unlock()
				return rpcErrorResponse(req, http.StatusConflict, "EntityAlreadyExists.User.Policy", "The user policy already exists."), nil
			}
		}
		t.userPolicy[userName] = append(policies, ramPolicyFixture{Name: policyName, Type: policyType})
		t.mu.Unlock()
		return demoreplay.JSONResponse(req, http.StatusOK, api.AttachRAMPolicyToUserResponse{RequestID: "req-ram-attach-user-policy"}), nil
	case "DetachPolicyFromUser":
		userName := strings.TrimSpace(query.Get("UserName"))
		policyName := strings.TrimSpace(query.Get("PolicyName"))
		policyType := strings.TrimSpace(query.Get("PolicyType"))
		if _, ok := findRAMUser(userName); !ok {
			return rpcErrorResponse(req, http.StatusNotFound, "EntityNotExist.User", "The specified RAM user does not exist."), nil
		}
		t.mu.Lock()
		policies := t.userPolicy[userName]
		filtered := make([]ramPolicyFixture, 0, len(policies))
		removed := false
		for _, p := range policies {
			if !removed && p.Name == policyName && p.Type == policyType {
				removed = true
				continue
			}
			filtered = append(filtered, p)
		}
		t.userPolicy[userName] = filtered
		t.mu.Unlock()
		if !removed {
			return rpcErrorResponse(req, http.StatusNotFound, "EntityNotExist.User.Policy", "The user policy does not exist."), nil
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.DetachRAMPolicyFromUserResponse{RequestID: "req-ram-detach-user-policy"}), nil
	case "DeleteUser":
		return demoreplay.JSONResponse(req, http.StatusOK, api.DeleteRAMUserResponse{RequestID: "req-ram-delete-user"}), nil
	case "CreateRole":
		return demoreplay.JSONResponse(req, http.StatusOK, api.CreateRAMRoleResponse{RequestID: "req-ram-create-role"}), nil
	case "AttachPolicyToRole":
		return demoreplay.JSONResponse(req, http.StatusOK, api.AttachRAMPolicyToRoleResponse{RequestID: "req-ram-attach-role-policy"}), nil
	case "DetachPolicyFromRole":
		return demoreplay.JSONResponse(req, http.StatusOK, api.DetachRAMPolicyFromRoleResponse{RequestID: "req-ram-detach-role-policy"}), nil
	case "DeleteRole":
		return demoreplay.JSONResponse(req, http.StatusOK, api.DeleteRAMRoleResponse{RequestID: "req-ram-delete-role"}), nil
	}
	return rpcErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported RAM replay action: %s", action)), nil
}

func (t *transport) handleRDS(req *http.Request, action string) (*http.Response, error) {
	query := req.URL.Query()
	switch action {
	case "DescribeRegions":
		regions := make([]api.RDSRegion, 0, len(demoRegions))
		for _, region := range demoRegions {
			regions = append(regions, api.RDSRegion{RegionID: region})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.DescribeRDSRegionsResponse{
			RequestID: "req-rds-regions",
			Regions:   api.RDSRegionList{RDSRegion: regions},
		}), nil
	case "DescribeDBInstances":
		region := strings.TrimSpace(query.Get("RegionId"))
		pageNumber := demoreplay.ParseInt(query.Get("PageNumber"), 1)
		pageSize := demoreplay.ParseInt(query.Get("PageSize"), 100)
		items := databasesForRegion(region)
		window := demoreplay.PageWindow(len(items), pageNumber, pageSize)
		pageItems := make([]api.RDSInstance, 0, window.End-window.Start)
		for _, item := range items[window.Start:window.End] {
			pageItems = append(pageItems, api.RDSInstance{
				DBInstanceID:        item.InstanceID,
				Engine:              item.Engine,
				EngineVersion:       item.EngineVersion,
				RegionID:            item.Region,
				ConnectionString:    item.Address,
				InstanceNetworkType: item.NetworkType,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.DescribeRDSInstancesResponse{
			RequestID:        "req-rds-instances",
			PageNumber:       pageNumber,
			PageRecordCount:  pageSize,
			TotalRecordCount: len(items),
			Items:            api.RDSInstanceList{DBInstance: pageItems},
		}), nil
	case "DescribeDatabases":
		instanceID := strings.TrimSpace(query.Get("DBInstanceId"))
		for _, item := range demoRDSInstances {
			if item.InstanceID != instanceID {
				continue
			}
			databases := make([]api.RDSDatabase, 0, len(item.DBNames))
			for _, dbName := range item.DBNames {
				databases = append(databases, api.RDSDatabase{DBName: dbName})
			}
			return demoreplay.JSONResponse(req, http.StatusOK, api.DescribeRDSDatabasesResponse{
				RequestID: "req-rds-databases",
				Databases: api.RDSDatabaseList{Database: databases},
			}), nil
		}
		return rpcErrorResponse(req, http.StatusNotFound, "InvalidDBInstance.NotFound", "Specified DB instance does not exist."), nil
	case "CreateAccount":
		return demoreplay.JSONResponse(req, http.StatusOK, api.CreateRDSAccountResponse{RequestID: "req-rds-create-account"}), nil
	case "GrantAccountPrivilege":
		return demoreplay.JSONResponse(req, http.StatusOK, api.GrantRDSAccountPrivilegeResponse{RequestID: "req-rds-grant-account"}), nil
	case "DeleteAccount":
		return demoreplay.JSONResponse(req, http.StatusOK, api.DeleteRDSAccountResponse{RequestID: "req-rds-delete-account"}), nil
	}
	return rpcErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported RDS replay action: %s", action)), nil
}

func (t *transport) handleSAS(req *http.Request, action string) (*http.Response, error) {
	switch action {
	case "DescribeSuspEvents":
		events := make([]api.SASSuspEvent, 0, len(demoSASEvents))
		for _, item := range demoSASEvents {
			events = append(events, api.SASSuspEvent{
				SecurityEventIDs:      item.ID,
				AlarmEventNameDisplay: item.Name,
				InstanceName:          item.Affected,
				EventStatus:           item.Status,
				LastTime:              item.Time,
				Details: []api.SASEventDetail{
					{NameDisplay: "调用的API", ValueDisplay: item.API},
					{NameDisplay: "调用IP", ValueDisplay: item.SourceIP},
					{NameDisplay: "AK", ValueDisplay: item.AccessKey},
				},
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.DescribeSASSuspEventsResponse{
			CurrentPage: 1,
			PageSize:    len(events),
			RequestID:   "req-sas-events",
			TotalCount:  len(events),
			Count:       len(events),
			SuspEvents:  events,
		}), nil
	case "HandleSecurityEvents":
		return demoreplay.JSONResponse(req, http.StatusOK, api.HandleSASSecurityEventsResponse{
			RequestID: "req-sas-handle-events",
			HandleSecurityEventsResponse: api.HandleSASSecurityEventsResponseItem{
				TaskID: 2026042001,
			},
		}), nil
	}
	return rpcErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported SAS replay action: %s", action)), nil
}

func (t *transport) handleDNS(req *http.Request, action string) (*http.Response, error) {
	query := req.URL.Query()
	switch action {
	case "DescribeDomains":
		pageNumber := demoreplay.ParseInt(query.Get("PageNumber"), 1)
		pageSize := demoreplay.ParseInt(query.Get("PageSize"), 100)
		window := demoreplay.PageWindow(len(demoDomains), pageNumber, pageSize)
		items := make([]api.DomainSummary, 0, window.End-window.Start)
		for _, domain := range demoDomains[window.Start:window.End] {
			items = append(items, api.DomainSummary{DomainName: domain.DomainName})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.DescribeDomainsResponse{
			TotalCount: len(demoDomains),
			PageSize:   pageSize,
			RequestID:  "req-dns-domains",
			PageNumber: pageNumber,
			Domains:    api.DomainList{Domain: items},
		}), nil
	case "DescribeDomainRecords":
		domain, ok := findDomain(query.Get("DomainName"))
		if !ok {
			return demoreplay.JSONResponse(req, http.StatusOK, api.DescribeDomainRecordsResponse{
				PageSize:      demoreplay.ParseInt(query.Get("PageSize"), 100),
				RequestID:     "req-dns-domain-records",
				PageNumber:    demoreplay.ParseInt(query.Get("PageNumber"), 1),
				DomainRecords: api.DomainRecordList{},
			}), nil
		}
		records := make([]api.DomainRecord, 0, len(domain.Records))
		for _, record := range domain.Records {
			records = append(records, api.DomainRecord{
				RR:     record.RR,
				Type:   record.Type,
				Value:  record.Value,
				Status: record.Status,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, api.DescribeDomainRecordsResponse{
			TotalCount:    len(records),
			PageSize:      demoreplay.ParseInt(query.Get("PageSize"), 100),
			RequestID:     "req-dns-domain-records",
			PageNumber:    demoreplay.ParseInt(query.Get("PageNumber"), 1),
			DomainRecords: api.DomainRecordList{Record: records},
		}), nil
	}
	return rpcErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported DNS replay action: %s", action)), nil
}

func (t *transport) handleSMS(req *http.Request, action string) (*http.Response, error) {
	switch action {
	case "QuerySmsSignList":
		resp := api.QuerySMSSignListResponse{
			RequestID:   "req-sms-signs",
			Code:        "OK",
			Message:     "OK",
			TotalCount:  1,
			CurrentPage: 1,
			PageSize:    10,
		}
		for _, sign := range demoSMSSigns() {
			resp.SmsSignList = append(resp.SmsSignList, api.SMSSignInfo{
				SignName:     sign["SignName"],
				AuditStatus:  sign["AuditStatus"],
				BusinessType: sign["BusinessType"],
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "QuerySmsTemplateList":
		resp := api.QuerySMSTemplateListResponse{
			RequestID:   "req-sms-templates",
			Code:        "OK",
			Message:     "OK",
			TotalCount:  1,
			CurrentPage: 1,
			PageSize:    10,
		}
		for _, template := range demoSMSTemplates() {
			resp.SmsTemplateList = append(resp.SmsTemplateList, api.SMSTemplateInfo{
				TemplateName:    template["TemplateName"],
				AuditStatus:     template["AuditStatus"],
				TemplateContent: template["TemplateContent"],
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "QuerySendStatistics":
		return demoreplay.JSONResponse(req, http.StatusOK, api.QuerySMSSendStatisticsResponse{
			RequestID: "req-sms-stats",
			Code:      "OK",
			Message:   "OK",
			Data: api.SMSSendStatisticsData{
				TotalSize: demoSMSDailySize(),
			},
		}), nil
	}
	return rpcErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported SMS replay action: %s", action)), nil
}

func (t *transport) handleOSS(req *http.Request) (*http.Response, error) {
	switch verifyOSSAuth(req) {
	case demoreplay.AuthInvalidAccessKey:
		return ossErrorResponse(req, http.StatusForbidden, "InvalidAccessKeyId", "The OSS Access Key Id you provided does not exist in our records."), nil
	case demoreplay.AuthInvalidSignature:
		return ossErrorResponse(req, http.StatusForbidden, "SignatureDoesNotMatch", "The request signature we calculated does not match the signature you provided."), nil
	}

	bucketName := bucketFromOSSHost(requestHost(req))
	if bucketName == "" {
		buckets := make([]oss.OSSBucket, 0, len(demoBuckets))
		for _, bucket := range demoBuckets {
			buckets = append(buckets, oss.OSSBucket{
				Name:     bucket.Name,
				Location: "oss-" + bucket.Region,
				Region:   bucket.Region,
			})
		}
		return demoreplay.XMLResponse(req, http.StatusOK, oss.ListBucketsResponse{
			MaxKeys: 1000,
			Buckets: buckets,
		}), nil
	}

	bucket, ok := findBucket(bucketName)
	if !ok {
		return ossErrorResponse(req, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist."), nil
	}
	if region := ossRegionFromHost(requestHost(req)); region != "" && region != bucket.Region {
		return ossErrorResponse(req, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist."), nil
	}
	query := req.URL.Query()
	if _, hasACL := query["acl"]; hasACL {
		return t.handleOSSBucketACL(req, bucket.Name)
	}
	maxKeys := demoreplay.ParseInt(query.Get("max-keys"), 1000)
	window := demoreplay.PageWindow(len(bucket.Objects), 1, maxKeys)
	return demoreplay.XMLResponse(req, http.StatusOK, oss.ListObjectsResponse{
		Name:        bucket.Name,
		MaxKeys:     maxKeys,
		IsTruncated: window.End < len(bucket.Objects),
		Objects:     append([]oss.OSSObject(nil), bucket.Objects[window.Start:window.End]...),
	}), nil
}

func (t *transport) handleOSSBucketACL(req *http.Request, bucketName string) (*http.Response, error) {
	switch req.Method {
	case http.MethodGet:
		t.mu.Lock()
		grant, ok := t.bucketACL[bucketName]
		t.mu.Unlock()
		if !ok {
			grant = "private"
		}
		resp := oss.BucketACLResponse{}
		resp.Owner.ID = "235000000000000001"
		resp.Owner.DisplayName = demoAccountAlias()
		resp.AccessControlList.Grant = grant
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	case http.MethodPut:
		acl := strings.TrimSpace(req.Header.Get("x-oss-acl"))
		if acl != "private" && acl != "public-read" && acl != "public-read-write" {
			return ossErrorResponse(req, http.StatusBadRequest, "InvalidArgument", "Unsupported x-oss-acl value."), nil
		}
		t.mu.Lock()
		t.bucketACL[bucketName] = acl
		t.mu.Unlock()
		return demoreplay.XMLResponse(req, http.StatusOK, struct{}{}), nil
	}
	return ossErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed", "Unsupported ACL method."), nil
}

func (t *transport) handleSLS(req *http.Request) (*http.Response, error) {
	switch verifySLSAuth(req) {
	case demoreplay.AuthInvalidAccessKey:
		return slsErrorResponse(req, http.StatusUnauthorized, "Unauthorized", "The provided AccessKeyId is invalid."), nil
	case demoreplay.AuthInvalidSignature:
		return slsErrorResponse(req, http.StatusUnauthorized, "Unauthorized", "The request signature we calculated does not match the signature you provided."), nil
	}

	region := slsRegionFromHost(requestHost(req))
	if req.Method == http.MethodGet && req.URL.Path == "/" {
		offset := demoreplay.ParseInt(req.URL.Query().Get("offset"), 0)
		size := demoreplay.ParseInt(req.URL.Query().Get("size"), 100)
		projects := logProjectsForRegion(region)
		window := demoreplay.OffsetWindow(len(projects), offset, size)
		count := int64(window.End - window.Start)
		total := int64(len(projects))
		resp := sls.ListProjectResponse{
			Count: &count,
			Total: &total,
		}
		for _, project := range projects[window.Start:window.End] {
			projectName := project.ProjectName
			projectRegion := project.Region
			description := project.Description
			lastModifyTime := fmt.Sprintf("%d", project.ModifiedAt.Unix())
			resp.Projects = append(resp.Projects, &sls.Project{
				ProjectName:    &projectName,
				Region:         &projectRegion,
				Description:    &description,
				LastModifyTime: &lastModifyTime,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}

	return slsErrorResponse(req, http.StatusNotFound, "NotFound", "Unsupported SLS replay request."), nil
}

func verifyRPCAuth(req *http.Request) demoreplay.AuthFailureKind {
	query := httpclient.CloneValues(req.URL.Query())
	accessKeyID := strings.TrimSpace(query.Get("AccessKeyId"))
	if accessKeyID != DemoAccessKeyID {
		return demoreplay.AuthInvalidAccessKey
	}
	signature := strings.TrimSpace(query.Get("Signature"))
	query.Del("Signature")
	expected := signRPC(req.Method, query, DemoAccessKeySecret+"&")
	if !demoreplay.SubtleEqual(signature, expected) {
		return demoreplay.AuthInvalidSignature
	}
	return demoreplay.AuthOK
}

func verifyOSSAuth(req *http.Request) demoreplay.AuthFailureKind {
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))
	accessKey, signature, ok := parseAuthorization(authHeader, "OSS ")
	if !ok {
		return demoreplay.AuthInvalidSignature
	}
	if accessKey != DemoAccessKeyID {
		return demoreplay.AuthInvalidAccessKey
	}
	clone := cloneRequest(req)
	clone.Header.Del("Authorization")
	if err := oss.Sign(clone, demoCredential(), bucketFromOSSHost(requestHost(req)), time.Time{}); err != nil {
		return demoreplay.AuthInvalidSignature
	}
	cloneAuthHeader := strings.TrimSpace(clone.Header.Get("Authorization"))
	_, expectedSignature, ok := parseAuthorization(cloneAuthHeader, "OSS ")
	if !ok || !demoreplay.SubtleEqual(signature, expectedSignature) {
		return demoreplay.AuthInvalidSignature
	}
	return demoreplay.AuthOK
}

func verifySLSAuth(req *http.Request) demoreplay.AuthFailureKind {
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))
	accessKey, signature, ok := parseAuthorization(authHeader, "LOG ")
	if !ok {
		return demoreplay.AuthInvalidSignature
	}
	if accessKey != DemoAccessKeyID {
		return demoreplay.AuthInvalidAccessKey
	}
	expected := signSLSRequest(req, DemoAccessKeySecret)
	if !demoreplay.SubtleEqual(signature, expected) {
		return demoreplay.AuthInvalidSignature
	}
	return demoreplay.AuthOK
}

func demoCredential() aliauth.Credential {
	return aliauth.New(DemoAccessKeyID, DemoAccessKeySecret, "")
}

func signRPC(method string, params url.Values, secret string) string {
	formed := params.Encode()
	formed = strings.ReplaceAll(formed, "+", "%20")
	formed = strings.ReplaceAll(formed, "*", "%2A")
	formed = strings.ReplaceAll(formed, "%7E", "~")
	method = strings.ToUpper(strings.TrimSpace(method))
	stringToSign := method
	if stringToSign == "" {
		stringToSign = http.MethodGet
	}
	stringToSign += "&%2F&" + url.QueryEscape(formed)
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func signSLSRequest(req *http.Request, secret string) string {
	contentMD5 := req.Header.Get("Content-MD5")
	contentType := req.Header.Get("Content-Type")
	date := req.Header.Get("Date")
	stringToSign := req.Method + "\n" + contentMD5 + "\n" + contentType + "\n" + date + "\n" + canonicalizeSLSHeaders(req.Header) + "\n" + canonicalizeSLSResource(req.URL)
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func canonicalizeSLSHeaders(headers http.Header) string {
	keys := make([]string, 0, len(headers))
	values := make(map[string]string, len(headers))
	for key, items := range headers {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		if !strings.HasPrefix(lowerKey, "x-log-") && !strings.HasPrefix(lowerKey, "x-acs-") {
			continue
		}
		keys = append(keys, lowerKey)
		if len(items) > 0 {
			values[lowerKey] = items[0]
		} else {
			values[lowerKey] = ""
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+":"+values[key])
	}
	return strings.Join(parts, "\n")
}

func canonicalizeSLSResource(u *url.URL) string {
	if u == nil {
		return "/"
	}
	resource := u.EscapedPath()
	if resource == "" {
		resource = "/"
	}
	if len(u.Query()) == 0 {
		return resource
	}
	keys := make([]string, 0, len(u.Query()))
	for key := range u.Query() {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		values := u.Query()[key]
		value := ""
		if len(values) > 0 {
			value = values[0]
		}
		parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
	}
	return resource + "?" + strings.Join(parts, "&")
}

func parseAuthorization(value, prefix string) (string, string, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, prefix) {
		return "", "", false
	}
	value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	accessKey := strings.TrimSpace(parts[0])
	signature := strings.TrimSpace(parts[1])
	if accessKey == "" || signature == "" {
		return "", "", false
	}
	return accessKey, signature, true
}

func cloneRequest(req *http.Request) *http.Request {
	if req == nil {
		return nil
	}
	clone := req.Clone(context.Background())
	if req.URL != nil {
		urlCopy := *req.URL
		clone.URL = &urlCopy
	}
	clone.Header = req.Header.Clone()
	clone.Host = req.Host
	return clone
}

func markerOffset(marker string) int {
	marker = strings.TrimSpace(marker)
	offset, err := strconv.Atoi(marker)
	if err != nil || offset < 0 {
		return 0
	}
	return offset
}

func buildCommandID(instanceID, command string) string {
	instanceID = strings.TrimSpace(instanceID)
	command = strings.TrimSpace(command)
	mac := hmac.New(sha1.New, []byte("ctk-replay"))
	_, _ = mac.Write([]byte(instanceID))
	_, _ = mac.Write([]byte{':'})
	_, _ = mac.Write([]byte(command))
	return fmt.Sprintf("c-%x", mac.Sum(nil)[:6])
}

func isOSSHost(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	host := strings.ToLower(req.URL.Hostname())
	return strings.HasPrefix(host, "oss-") || strings.Contains(host, ".oss-")
}

func bucketFromOSSHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimSuffix(host, ":443")
	if strings.HasPrefix(host, "oss-") {
		return ""
	}
	parts := strings.SplitN(host, ".oss-", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func ossRegionFromHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimSuffix(host, ":443")
	switch {
	case strings.HasPrefix(host, "oss-"):
		return strings.TrimSuffix(strings.TrimPrefix(host, "oss-"), ".aliyuncs.com")
	case strings.Contains(host, ".oss-"):
		parts := strings.SplitN(host, ".oss-", 2)
		if len(parts) != 2 {
			return ""
		}
		return strings.TrimSuffix(parts[1], ".aliyuncs.com")
	default:
		return ""
	}
}

func isSLSHost(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	return strings.Contains(strings.ToLower(req.URL.Hostname()), ".log.aliyuncs.com")
}

func rpcProductFromHost(host string) string {
	host = normalizeRPCReplayHost(host)
	switch {
	case host == "sts.aliyuncs.com" || strings.HasPrefix(host, "sts-vpc."):
		return "sts"
	case host == "location-readonly.aliyuncs.com":
		return "location"
	case host == "business.aliyuncs.com" || strings.HasPrefix(host, "business."):
		return "bssopenapi"
	case host == "ram.aliyuncs.com":
		return "ram"
	case host == "alidns.aliyuncs.com":
		return "alidns"
	case host == "dysmsapi.aliyuncs.com":
		return "dysmsapi"
	case host == "tds.aliyuncs.com":
		return "sas"
	case isECSRPCHost(host):
		return "ecs"
	case isRDSRPCHost(host):
		return "rds"
	default:
		return ""
	}
}

func isECSRPCHost(host string) bool {
	host = normalizeRPCReplayHost(host)
	return host == "ecs.aliyuncs.com" ||
		strings.HasPrefix(host, "ecs-") ||
		strings.HasPrefix(host, "ecs.") ||
		strings.HasPrefix(host, "ecs-vpc.")
}

func isRDSRPCHost(host string) bool {
	host = normalizeRPCReplayHost(host)
	return host == "rds.aliyuncs.com" ||
		strings.HasPrefix(host, "rds-") ||
		strings.HasPrefix(host, "rds.") ||
		strings.HasPrefix(host, "rds-vpc.")
}

func replayECSEndpoint(region string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		region = api.DefaultRegion
	}
	switch {
	case strings.HasPrefix(region, "cn-"):
		return "ecs-" + region + ".aliyuncs.com"
	default:
		return "ecs." + region + ".aliyuncs.com"
	}
}

func normalizeRPCReplayHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimSuffix(host, ":443")
	host = strings.TrimSuffix(host, ":80")
	return host
}

func requestHost(req *http.Request) string {
	if req == nil {
		return ""
	}
	if value := strings.TrimSpace(req.Host); value != "" {
		return value
	}
	if req.URL != nil {
		return req.URL.Host
	}
	return ""
}

func slsRegionFromHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimSuffix(host, ":443")
	prefix := strings.TrimSuffix(host, ".log.aliyuncs.com")
	if prefix == host {
		return ""
	}
	parts := strings.Split(prefix, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func rpcErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	return demoreplay.JSONResponse(req, statusCode, map[string]string{
		"Code":      code,
		"Message":   message,
		"RequestId": "req-replay-auth",
	})
}

func ossErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	return demoreplay.XMLResponse(req, statusCode, ossErrorEnvelope{
		Code:      code,
		Message:   message,
		RequestID: "req-replay-oss",
		HostID:    "oss-replay.aliyuncs.com",
	})
}

func slsErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	return demoreplay.JSONResponse(req, statusCode, map[string]string{
		"errorCode":    code,
		"errorMessage": message,
	})
}

type ossErrorEnvelope struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	RequestID string   `xml:"RequestId"`
	HostID    string   `xml:"HostId"`
}
