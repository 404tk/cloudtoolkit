package replay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cos"
)

type invocationResult struct {
	CommandID    string
	InvocationID string
	TaskID       string
	InstanceID   string
	Output       string
}

type transport struct {
	mu           sync.Mutex
	sequence     int
	createdUsers map[string]camUserFixture
	userPolicies map[uint64][]camPolicyFixture
	bucketACL    map[string]string
	invocations  map[string]invocationResult
	tasks        map[string]invocationResult
}

func newTransport() *transport {
	return &transport{
		createdUsers: make(map[string]camUserFixture),
		userPolicies: seedUserPolicies(),
		bucketACL:    seedTencentBucketACL(),
		invocations:  make(map[string]invocationResult),
		tasks:        make(map[string]invocationResult),
	}
}

// seedUserPolicies copies fixture policy attachments out of demoCAMUsers so
// the transport can mutate them per request without altering shared fixtures.
func seedUserPolicies() map[uint64][]camPolicyFixture {
	out := make(map[uint64][]camPolicyFixture, len(demoCAMUsers))
	for _, user := range demoCAMUsers {
		policies := make([]camPolicyFixture, len(user.Policies))
		copy(policies, user.Policies)
		out[user.UIN] = policies
	}
	return out
}

// seedTencentBucketACL gives each demo COS bucket a starting "private" canned
// ACL so audit/expose/audit/unexpose cycles surface deterministic state.
func seedTencentBucketACL() map[string]string {
	out := make(map[string]string, len(demoBuckets))
	for _, bucket := range demoBuckets {
		out[bucket.Name] = "private"
	}
	return out
}

func findPolicyByID(id uint64) (camPolicyFixture, bool) {
	for _, p := range demoPolicies {
		if p.ID == id {
			return p, true
		}
	}
	return camPolicyFixture{}, false
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := demoreplay.ReadRequestBody(req)
	if err != nil {
		return nil, err
	}
	if isCOSHost(req.URL.Hostname()) {
		return t.handleCOS(req)
	}
	return t.handleOpenAPI(req, body)
}

func (t *transport) handleOpenAPI(req *http.Request, body []byte) (*http.Response, error) {
	switch verifyOpenAPIAuth(req, body) {
	case demoreplay.AuthInvalidAccessKey:
		return openAPIErrorResponse(req, http.StatusForbidden, "AuthFailure.SecretIdNotFound", "The SecretId is not found."), nil
	case demoreplay.AuthInvalidSignature:
		return openAPIErrorResponse(req, http.StatusForbidden, "AuthFailure.SignatureFailure", "The provided credentials could not be validated. Please check your SecretId and SecretKey."), nil
	}

	service := requestService(req)
	action := tcAction(req)
	switch service {
	case "sts":
		return t.handleSTS(req, action)
	case "billing":
		return t.handleBilling(req, action)
	case "cam":
		return t.handleCAM(req, action, body)
	case "cvm":
		return t.handleCVM(req, action, body)
	case "lighthouse":
		return t.handleLighthouse(req, action, body)
	case "dnspod":
		return t.handleDNSPod(req, action, body)
	case "cdb":
		return t.handleCDB(req, action)
	case "mariadb":
		return t.handleMariaDB(req, action)
	case "postgres":
		return t.handlePostgres(req, action)
	case "sqlserver":
		return t.handleSQLServer(req, action)
	case "tat":
		return t.handleTAT(req, action, body)
	case "cloudaudit":
		return t.handleCloudAudit(req, action)
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
}

func (t *transport) handleSTS(req *http.Request, action string) (*http.Response, error) {
	if action != "GetCallerIdentity" {
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
	resp := api.GetCallerIdentityResponse{}
	resp.Response.Arn = demoCallerID
	resp.Response.Type = "root"
	resp.Response.UserID = demoOwnerUIN
	resp.Response.RequestID = "req-replay-sts"
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleBilling(req *http.Request, action string) (*http.Response, error) {
	if action != "DescribeAccountBalance" {
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
	balance := demoBalanceCents()
	resp := api.DescribeAccountBalanceResponse{}
	resp.Response.Balance = &balance
	resp.Response.RequestID = "req-replay-billing"
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleCAM(req *http.Request, action string, body []byte) (*http.Response, error) {
	switch action {
	case "ListUsers":
		users := listCAMUsers(t.snapshotUsers())
		sort.Slice(users, func(i, j int) bool { return users[i].UIN < users[j].UIN })
		resp := api.ListUsersResponse{}
		resp.Response.Data = make([]api.SubAccountInfo, 0, len(users))
		resp.Response.RequestID = "req-replay-cam-list-users"
		for _, user := range users {
			uin := user.UIN
			name := user.Name
			createTime := user.CreateTime
			consoleLogin := uint64(0)
			if user.ConsoleLogin {
				consoleLogin = 1
			}
			resp.Response.Data = append(resp.Response.Data, api.SubAccountInfo{
				Uin:          &uin,
				Name:         &name,
				ConsoleLogin: &consoleLogin,
				CreateTime:   &createTime,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "ListAttachedUserAllPolicies":
		var payload api.ListAttachedUserAllPoliciesRequest
		_ = json.Unmarshal(body, &payload)
		uin := derefUint64(payload.TargetUin)
		if _, ok := t.findUserByUIN(uin); !ok {
			return openAPIErrorResponse(req, http.StatusNotFound, "ResourceNotFound.User", "The specified user does not exist."), nil
		}
		t.mu.Lock()
		policies := append([]camPolicyFixture(nil), t.userPolicies[uin]...)
		t.mu.Unlock()
		resp := api.ListAttachedUserAllPoliciesResponse{}
		resp.Response.PolicyList = make([]api.AttachedUserPolicy, 0, len(policies))
		resp.Response.RequestID = "req-replay-cam-list-policies"
		total := uint64(len(policies))
		resp.Response.TotalNum = &total
		for _, policy := range policies {
			id := fmt.Sprintf("%d", policy.ID)
			name := policy.Name
			strategyType := policy.StrategyType
			resp.Response.PolicyList = append(resp.Response.PolicyList, api.AttachedUserPolicy{
				PolicyID:     &id,
				PolicyName:   &name,
				StrategyType: &strategyType,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "GetPolicy":
		var payload api.GetPolicyRequest
		_ = json.Unmarshal(body, &payload)
		policy, ok := findPolicy(derefUint64(payload.PolicyID))
		if !ok {
			return openAPIErrorResponse(req, http.StatusNotFound, "ResourceNotFound.Policy", "The specified policy does not exist."), nil
		}
		resp := api.GetPolicyResponse{}
		resp.Response.PolicyDocument = stringPtr(policy.Document)
		resp.Response.RequestID = "req-replay-cam-get-policy"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "AddUser":
		var payload api.AddUserRequest
		_ = json.Unmarshal(body, &payload)
		user := t.ensureUser(derefString(payload.Name), derefString(payload.Password))
		resp := api.AddUserResponse{}
		resp.Response.Uin = uint64Ptr(user.UIN)
		resp.Response.Name = stringPtr(user.Name)
		resp.Response.Password = stringPtr(derefString(payload.Password))
		resp.Response.RequestID = "req-replay-cam-add-user"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "GetUser":
		var payload api.GetUserRequest
		_ = json.Unmarshal(body, &payload)
		user, ok := t.findUserByName(derefString(payload.Name))
		if !ok {
			return openAPIErrorResponse(req, http.StatusNotFound, "ResourceNotFound.User", "The specified user does not exist."), nil
		}
		resp := api.GetUserResponse{}
		resp.Response.Uin = uint64Ptr(user.UIN)
		resp.Response.Name = stringPtr(user.Name)
		resp.Response.RequestID = "req-replay-cam-get-user"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "AttachUserPolicy":
		var payload api.AttachUserPolicyRequest
		_ = json.Unmarshal(body, &payload)
		uin := derefUint64(payload.AttachUin)
		policyID := derefUint64(payload.PolicyID)
		if uin == 0 || policyID == 0 {
			return openAPIErrorResponse(req, http.StatusBadRequest, "InvalidParameter", "AttachUin and PolicyId required"), nil
		}
		if _, ok := t.findUserByUIN(uin); !ok {
			return openAPIErrorResponse(req, http.StatusNotFound, "ResourceNotFound.User", "The specified user does not exist."), nil
		}
		policy, ok := findPolicyByID(policyID)
		if !ok {
			policy = camPolicyFixture{ID: policyID, Name: fmt.Sprintf("policy-%d", policyID), StrategyType: "1"}
		}
		t.mu.Lock()
		existing := t.userPolicies[uin]
		for _, p := range existing {
			if p.ID == policyID {
				t.mu.Unlock()
				resp := api.AttachUserPolicyResponse{}
				resp.Response.RequestID = "req-replay-cam-attach-user-policy"
				return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
			}
		}
		t.userPolicies[uin] = append(existing, policy)
		t.mu.Unlock()
		resp := api.AttachUserPolicyResponse{}
		resp.Response.RequestID = "req-replay-cam-attach-user-policy"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DetachUserPolicy":
		var payload api.DetachUserPolicyRequest
		_ = json.Unmarshal(body, &payload)
		uin := derefUint64(payload.DetachUin)
		policyID := derefUint64(payload.PolicyID)
		if uin == 0 || policyID == 0 {
			return openAPIErrorResponse(req, http.StatusBadRequest, "InvalidParameter", "DetachUin and PolicyId required"), nil
		}
		if _, ok := t.findUserByUIN(uin); !ok {
			return openAPIErrorResponse(req, http.StatusNotFound, "ResourceNotFound.User", "The specified user does not exist."), nil
		}
		t.mu.Lock()
		existing := t.userPolicies[uin]
		filtered := make([]camPolicyFixture, 0, len(existing))
		removed := false
		for _, p := range existing {
			if !removed && p.ID == policyID {
				removed = true
				continue
			}
			filtered = append(filtered, p)
		}
		t.userPolicies[uin] = filtered
		t.mu.Unlock()
		resp := api.DetachUserPolicyResponse{}
		resp.Response.RequestID = "req-replay-cam-detach-user-policy"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "GetUserAppId":
		resp := api.GetUserAppIDResponse{}
		resp.Response.OwnerUin = stringPtr(demoOwnerUIN)
		resp.Response.RequestID = "req-replay-cam-get-user-appid"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DeleteUser":
		var payload api.DeleteUserRequest
		_ = json.Unmarshal(body, &payload)
		t.deleteUser(derefString(payload.Name))
		resp := api.DeleteUserResponse{}
		resp.Response.RequestID = "req-replay-cam-delete-user"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "CreateRole":
		resp := api.CreateRoleResponse{}
		resp.Response.RoleID = stringPtr("qcs::cam::roleName/ctk-demo-role")
		resp.Response.RequestID = "req-replay-cam-create-role"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "AttachRolePolicy":
		resp := api.AttachRolePolicyResponse{}
		resp.Response.RequestID = "req-replay-cam-attach-role-policy"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DetachRolePolicy":
		resp := api.DetachRolePolicyResponse{}
		resp.Response.RequestID = "req-replay-cam-detach-role-policy"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DeleteRole":
		resp := api.DeleteRoleResponse{}
		resp.Response.RequestID = "req-replay-cam-delete-role"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
}

func (t *transport) handleCVM(req *http.Request, action string, body []byte) (*http.Response, error) {
	switch action {
	case "DescribeRegions":
		resp := api.DescribeCVMRegionsResponse{}
		resp.Response.RegionSet = make([]api.CVMRegionInfo, 0, len(demoRegions))
		resp.Response.RequestID = "req-replay-cvm-regions"
		for _, region := range demoRegions {
			r := region
			resp.Response.RegionSet = append(resp.Response.RegionSet, api.CVMRegionInfo{Region: &r})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeInstances":
		var payload api.DescribeCVMInstancesRequest
		_ = json.Unmarshal(body, &payload)
		region := tcRegion(req)
		items := cvmForRegion(region)
		w := demoreplay.OffsetWindow(len(items), int(derefInt64(payload.Offset)), int(derefInt64(payload.Limit, 100)))
		start, end := w.Start, w.End
		resp := api.DescribeCVMInstancesResponse{}
		total := int64(len(items))
		resp.Response.TotalCount = &total
		resp.Response.InstanceSet = make([]api.CVMInstanceInfo, 0, end-start)
		resp.Response.RequestID = "req-replay-cvm-instances"
		for _, item := range items[start:end] {
			instanceID := item.InstanceID
			instanceName := item.InstanceName
			state := item.State
			osName := item.OSName
			resp.Response.InstanceSet = append(resp.Response.InstanceSet, api.CVMInstanceInfo{
				InstanceID:         &instanceID,
				InstanceName:       &instanceName,
				InstanceState:      &state,
				PublicIPAddresses:  demoreplay.NonEmptyStrings(item.PublicIP),
				PrivateIPAddresses: demoreplay.NonEmptyStrings(item.PrivateIP),
				OSName:             &osName,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
}

func (t *transport) handleLighthouse(req *http.Request, action string, body []byte) (*http.Response, error) {
	switch action {
	case "DescribeRegions":
		resp := api.DescribeLighthouseRegionsResponse{}
		resp.Response.RegionSet = make([]api.LighthouseRegionInfo, 0, len(demoRegions))
		resp.Response.RequestID = "req-replay-lighthouse-regions"
		for _, region := range demoRegions {
			r := region
			resp.Response.RegionSet = append(resp.Response.RegionSet, api.LighthouseRegionInfo{Region: &r})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeInstances":
		var payload api.DescribeLighthouseInstancesRequest
		_ = json.Unmarshal(body, &payload)
		region := tcRegion(req)
		items := lighthouseForRegion(region)
		w := demoreplay.OffsetWindow(len(items), int(derefInt64(payload.Offset)), int(derefInt64(payload.Limit, 100)))
		start, end := w.Start, w.End
		resp := api.DescribeLighthouseInstancesResponse{}
		total := int64(len(items))
		resp.Response.TotalCount = &total
		resp.Response.InstanceSet = make([]api.LighthouseInstanceInfo, 0, end-start)
		resp.Response.RequestID = "req-replay-lighthouse-instances"
		for _, item := range items[start:end] {
			instanceID := item.InstanceID
			instanceName := item.InstanceName
			state := item.State
			platformType := item.PlatformType
			resp.Response.InstanceSet = append(resp.Response.InstanceSet, api.LighthouseInstanceInfo{
				InstanceID:       &instanceID,
				InstanceName:     &instanceName,
				InstanceState:    &state,
				PublicAddresses:  demoreplay.NonEmptyStrings(item.PublicAddress),
				PrivateAddresses: demoreplay.NonEmptyStrings(item.PrivateIP),
				PlatformType:     &platformType,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
}

func (t *transport) handleDNSPod(req *http.Request, action string, body []byte) (*http.Response, error) {
	switch action {
	case "DescribeDomainList":
		var payload api.DescribeDomainListRequest
		_ = json.Unmarshal(body, &payload)
		w := demoreplay.OffsetWindow(len(demoDomains), int(payload.Offset), int(payload.Limit))
		start, end := w.Start, w.End
		resp := api.DescribeDomainListResponse{}
		total := uint64(len(demoDomains))
		resp.Response.DomainCountInfo.DomainTotal = &total
		resp.Response.DomainList = make([]api.DomainListItem, 0, end-start)
		resp.Response.RequestID = "req-replay-dnspod-domains"
		for _, domain := range demoDomains[start:end] {
			name := domain.Name
			status := domain.Status
			dnsStatus := domain.DNSStatus
			resp.Response.DomainList = append(resp.Response.DomainList, api.DomainListItem{
				Name:      &name,
				Status:    &status,
				DNSStatus: &dnsStatus,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeRecordList":
		var payload api.DescribeRecordListRequest
		_ = json.Unmarshal(body, &payload)
		domain, ok := findDomain(payload.Domain)
		if !ok {
			return openAPIErrorResponse(req, http.StatusNotFound, "ResourceNotFound.Domain", "The specified domain does not exist."), nil
		}
		w := demoreplay.OffsetWindow(len(domain.Records), int(payload.Offset), int(payload.Limit))
		start, end := w.Start, w.End
		resp := api.DescribeRecordListResponse{}
		total := uint64(len(domain.Records))
		listCount := uint64(end - start)
		resp.Response.RecordCountInfo.TotalCount = &total
		resp.Response.RecordCountInfo.ListCount = &listCount
		resp.Response.RecordList = make([]api.RecordListItem, 0, end-start)
		resp.Response.RequestID = "req-replay-dnspod-records"
		for _, record := range domain.Records[start:end] {
			name := record.Name
			recordType := record.Type
			value := record.Value
			status := record.Status
			resp.Response.RecordList = append(resp.Response.RecordList, api.RecordListItem{
				Name:   &name,
				Type:   &recordType,
				Value:  &value,
				Status: &status,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
}

func (t *transport) handleCDB(req *http.Request, action string) (*http.Response, error) {
	switch action {
	case "DescribeCdbZoneConfig":
		resp := api.DescribeCDBZoneConfigResponse{}
		resp.Response.DataResult.Regions = make([]api.CDBRegionSellConf, 0, len(demoRegions))
		resp.Response.RequestID = "req-replay-cdb-regions"
		for _, region := range demoRegions {
			r := region
			resp.Response.DataResult.Regions = append(resp.Response.DataResult.Regions, api.CDBRegionSellConf{Region: &r})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeDBInstances":
		region := tcRegion(req)
		items := mysqlForRegion(region)
		resp := api.DescribeCDBInstancesResponse{}
		resp.Response.Items = make([]api.CDBInstanceInfo, 0, len(items))
		resp.Response.RequestID = "req-replay-cdb-instances"
		for _, item := range items {
			instanceID := item.InstanceID
			version := item.Version
			itemRegion := item.Region
			instance := api.CDBInstanceInfo{
				InstanceID:    &instanceID,
				EngineVersion: &version,
				Region:        &itemRegion,
				WanStatus:     int64Ptr(item.WanStatus),
			}
			if item.WanDomain != "" {
				instance.WanDomain = stringPtr(item.WanDomain)
				instance.WanPort = int64Ptr(item.WanPort)
			}
			if item.VIP != "" {
				instance.Vip = stringPtr(item.VIP)
				instance.Vport = int64Ptr(item.VPort)
			}
			resp.Response.Items = append(resp.Response.Items, instance)
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
}

func (t *transport) handleMariaDB(req *http.Request, action string) (*http.Response, error) {
	switch action {
	case "DescribeSaleInfo":
		resp := api.DescribeMariaDBSaleInfoResponse{}
		resp.Response.RegionList = make([]api.MariaDBRegionInfo, 0, len(demoRegions))
		resp.Response.RequestID = "req-replay-mariadb-regions"
		for _, region := range demoRegions {
			r := region
			resp.Response.RegionList = append(resp.Response.RegionList, api.MariaDBRegionInfo{Region: &r})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeDBInstances":
		region := tcRegion(req)
		items := mariadbForRegion(region)
		resp := api.DescribeMariaDBInstancesResponse{}
		resp.Response.Instances = make([]api.MariaDBInstanceInfo, 0, len(items))
		resp.Response.RequestID = "req-replay-mariadb-instances"
		for _, item := range items {
			instanceID := item.InstanceID
			version := item.Version
			itemRegion := item.Region
			instance := api.MariaDBInstanceInfo{
				InstanceID: &instanceID,
				DBVersion:  &version,
				Region:     &itemRegion,
				WanStatus:  int64Ptr(item.WanStatus),
			}
			if item.WanDomain != "" {
				instance.WanDomain = stringPtr(item.WanDomain)
				instance.WanPort = int64Ptr(item.WanPort)
			}
			if item.VIP != "" {
				instance.Vip = stringPtr(item.VIP)
				instance.Vport = int64Ptr(item.VPort)
			}
			resp.Response.Instances = append(resp.Response.Instances, instance)
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
}

func (t *transport) handlePostgres(req *http.Request, action string) (*http.Response, error) {
	switch action {
	case "DescribeRegions":
		resp := api.DescribePostgresRegionsResponse{}
		resp.Response.RegionSet = make([]api.PostgresRegionInfo, 0, len(demoRegions))
		resp.Response.RequestID = "req-replay-postgres-regions"
		for _, region := range demoRegions {
			r := region
			state := "AVAILABLE"
			resp.Response.RegionSet = append(resp.Response.RegionSet, api.PostgresRegionInfo{
				Region:      &r,
				RegionState: &state,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeDBInstances":
		region := tcRegion(req)
		items := postgresForRegion(region)
		resp := api.DescribePostgresInstancesResponse{}
		resp.Response.DBInstanceSet = make([]api.PostgresInstanceInfo, 0, len(items))
		resp.Response.RequestID = "req-replay-postgres-instances"
		for _, item := range items {
			instanceID := item.InstanceID
			engine := item.Engine
			version := item.Version
			itemRegion := item.Region
			publicNetType := "public"
			privateNetType := "private"
			opened := "opened"
			privateIP := item.PrivateIP
			publicAddress := item.PublicAddress
			resp.Response.DBInstanceSet = append(resp.Response.DBInstanceSet, api.PostgresInstanceInfo{
				DBInstanceID:      &instanceID,
				DBEngine:          &engine,
				DBInstanceVersion: &version,
				Region:            &itemRegion,
				DBInstanceNetInfo: []api.PostgresNetInfo{
					{
						IP:      &privateIP,
						Port:    uint64Ptr(item.Port),
						NetType: &privateNetType,
						Status:  &opened,
					},
					{
						Address: &publicAddress,
						Port:    uint64Ptr(item.Port),
						NetType: &publicNetType,
						Status:  &opened,
					},
				},
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
}

func (t *transport) handleSQLServer(req *http.Request, action string) (*http.Response, error) {
	switch action {
	case "DescribeRegions":
		resp := api.DescribeSQLServerRegionsResponse{}
		resp.Response.RegionSet = make([]api.SQLServerRegionInfo, 0, len(demoRegions))
		resp.Response.RequestID = "req-replay-sqlserver-regions"
		for _, region := range demoRegions {
			r := region
			state := "AVAILABLE"
			resp.Response.RegionSet = append(resp.Response.RegionSet, api.SQLServerRegionInfo{
				Region:      &r,
				RegionState: &state,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeDBInstances":
		region := tcRegion(req)
		items := sqlServerForRegion(region)
		resp := api.DescribeSQLServerInstancesResponse{}
		resp.Response.DBInstances = make([]api.SQLServerInstanceInfo, 0, len(items))
		resp.Response.RequestID = "req-replay-sqlserver-instances"
		for _, item := range items {
			instanceID := item.InstanceID
			versionName := item.VersionName
			version := item.Version
			itemRegion := item.Region
			instance := api.SQLServerInstanceInfo{
				InstanceID:  &instanceID,
				VersionName: &versionName,
				Version:     &version,
				Region:      &itemRegion,
			}
			if item.DNSPodDomain != "" {
				instance.DNSPodDomain = stringPtr(item.DNSPodDomain)
				instance.TgwWanVPort = int64Ptr(item.TgwWanVPort)
			}
			if item.VIP != "" {
				instance.Vip = stringPtr(item.VIP)
				instance.Vport = int64Ptr(item.VPort)
			}
			resp.Response.DBInstances = append(resp.Response.DBInstances, instance)
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
}

func (t *transport) handleTAT(req *http.Request, action string, body []byte) (*http.Response, error) {
	switch action {
	case "RunCommand":
		var payload api.RunTATCommandRequest
		_ = json.Unmarshal(body, &payload)
		instanceID := ""
		if len(payload.InstanceIDs) > 0 {
			instanceID = payload.InstanceIDs[0]
		}
		command := decodeCommandContent(derefString(payload.Content))
		result := t.newInvocation(instanceID, shellOutput(instanceID, command))
		resp := api.RunTATCommandResponse{}
		resp.Response.CommandID = stringPtr(result.CommandID)
		resp.Response.InvocationID = stringPtr(result.InvocationID)
		resp.Response.RequestID = "req-replay-tat-run-command"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeInvocations":
		var payload api.DescribeTATInvocationsRequest
		_ = json.Unmarshal(body, &payload)
		resp := api.DescribeTATInvocationsResponse{}
		resp.Response.InvocationSet = make([]api.TATInvocation, 0, len(payload.InvocationIDs))
		resp.Response.RequestID = "req-replay-tat-invocations"
		for _, invocationID := range payload.InvocationIDs {
			result, ok := t.findInvocation(invocationID)
			if !ok {
				continue
			}
			status := "RUNNING"
			taskID := result.TaskID
			instanceID := result.InstanceID
			resp.Response.InvocationSet = append(resp.Response.InvocationSet, api.TATInvocation{
				InvocationID: stringPtr(result.InvocationID),
				InvocationTaskBasicInfoSet: []api.TATInvocationTaskBasic{
					{
						InvocationTaskID: &taskID,
						TaskStatus:       &status,
						InstanceID:       &instanceID,
					},
				},
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeInvocationTasks":
		var payload api.DescribeTATInvocationTasksRequest
		_ = json.Unmarshal(body, &payload)
		resp := api.DescribeTATInvocationTasksResponse{}
		resp.Response.InvocationTaskSet = make([]api.TATInvocationTask, 0, len(payload.InvocationTaskIDs))
		resp.Response.RequestID = "req-replay-tat-invocation-tasks"
		for _, taskID := range payload.InvocationTaskIDs {
			result, ok := t.findTask(taskID)
			if !ok {
				continue
			}
			status := "SUCCESS"
			output := base64.StdEncoding.EncodeToString([]byte(result.Output))
			exitCode := int64(0)
			instanceID := result.InstanceID
			resp.Response.InvocationTaskSet = append(resp.Response.InvocationTaskSet, api.TATInvocationTask{
				InvocationID:     stringPtr(result.InvocationID),
				InvocationTaskID: stringPtr(result.TaskID),
				TaskStatus:       &status,
				InstanceID:       &instanceID,
				TaskResult: &api.TATTaskResult{
					ExitCode: &exitCode,
					Output:   &output,
				},
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
	}
}

func (t *transport) handleCOS(req *http.Request) (*http.Response, error) {
	switch verifyCOSAuth(req) {
	case demoreplay.AuthInvalidAccessKey:
		return xmlErrorResponse(req, http.StatusForbidden, "InvalidAccessKeyId", "The Access Key Id you provided does not exist in our records."), nil
	case demoreplay.AuthInvalidSignature:
		return xmlErrorResponse(req, http.StatusForbidden, "SignatureDoesNotMatch", "The request signature we calculated does not match the signature you provided."), nil
	}

	host := strings.ToLower(req.URL.Hostname())
	if host == "service.cos.myqcloud.com" && req.Method == http.MethodGet {
		resp := cos.ListBucketsResponse{
			Buckets: make([]cos.COSBucket, 0, len(demoBuckets)),
		}
		for _, bucket := range demoBuckets {
			resp.Buckets = append(resp.Buckets, cos.COSBucket{
				Name:         bucket.Name,
				Region:       bucket.Region,
				CreationDate: bucket.CreationDate,
			})
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	}

	bucketName, region, ok := parseBucketHost(host)
	if !ok {
		return xmlErrorResponse(req, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist."), nil
	}
	bucket, found := findBucket(bucketName)
	if !found || bucket.Region != region {
		return xmlErrorResponse(req, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist."), nil
	}

	if req.URL.Query().Has("acl") {
		return t.handleCOSBucketACL(req, bucket.Name)
	}

	switch req.Method {
	case http.MethodGet:
		maxKeys := demoreplay.ParseInt(req.URL.Query().Get("max-keys"), 1000)
		marker := strings.TrimSpace(req.URL.Query().Get("marker"))
		objects, nextMarker, truncated := bucketPage(bucket.Objects, marker, maxKeys)
		resp := cos.ListObjectsResponse{
			Name:        bucket.Name,
			Marker:      marker,
			NextMarker:  nextMarker,
			MaxKeys:     maxKeys,
			IsTruncated: truncated,
			Objects:     make([]cos.COSObject, 0, len(objects)),
		}
		for _, object := range objects {
			resp.Objects = append(resp.Objects, cos.COSObject{
				Key:  object.Key,
				Size: object.Size,
			})
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	default:
		return xmlErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed", "The specified method is not allowed against this resource."), nil
	}
}

func (t *transport) handleCOSBucketACL(req *http.Request, bucketName string) (*http.Response, error) {
	switch req.Method {
	case http.MethodGet:
		t.mu.Lock()
		acl, ok := t.bucketACL[bucketName]
		t.mu.Unlock()
		if !ok {
			acl = "private"
		}
		resp := cos.BucketACLResponse{}
		resp.Owner.ID = "qcs::cam::uin/100000001:uin/100000001"
		resp.Owner.DisplayName = "ctk-demo"
		switch acl {
		case "public-read":
			resp.AccessControlList.Grant = append(resp.AccessControlList.Grant, struct {
				Grantee struct {
					Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
					ID   string `xml:"ID"`
					URI  string `xml:"URI"`
				} `xml:"Grantee"`
				Permission string `xml:"Permission"`
			}{
				Grantee: struct {
					Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
					ID   string `xml:"ID"`
					URI  string `xml:"URI"`
				}{Type: "Group", URI: "http://cam.qcloud.com/groups/global/AllUsers"},
				Permission: "READ",
			})
		case "public-read-write":
			resp.AccessControlList.Grant = append(resp.AccessControlList.Grant, struct {
				Grantee struct {
					Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
					ID   string `xml:"ID"`
					URI  string `xml:"URI"`
				} `xml:"Grantee"`
				Permission string `xml:"Permission"`
			}{
				Grantee: struct {
					Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
					ID   string `xml:"ID"`
					URI  string `xml:"URI"`
				}{Type: "Group", URI: "http://cam.qcloud.com/groups/global/AllUsers"},
				Permission: "FULL_CONTROL",
			})
		}
		return demoreplay.XMLResponse(req, http.StatusOK, resp), nil
	case http.MethodPut:
		acl := strings.TrimSpace(req.Header.Get("x-cos-acl"))
		if acl == "" {
			return xmlErrorResponse(req, http.StatusBadRequest, "InvalidArgument", "missing x-cos-acl header"), nil
		}
		t.mu.Lock()
		t.bucketACL[bucketName] = acl
		t.mu.Unlock()
		return demoreplay.XMLResponse(req, http.StatusOK, struct{}{}), nil
	}
	return xmlErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed", "unsupported acl method"), nil
}

func verifyOpenAPIAuth(req *http.Request, body []byte) demoreplay.AuthFailureKind {
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))
	secretID, service, ok := parseTC3Credential(authHeader)
	if !ok {
		return demoreplay.AuthInvalidSignature
	}
	if !demoreplay.SubtleEqual(secretID, demoCredentials.AccessKey) {
		return demoreplay.AuthInvalidAccessKey
	}
	timestamp, err := parseUnixHeader(req.Header.Get("X-TC-Timestamp"))
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	host := strings.TrimSpace(req.Host)
	if host == "" {
		host = req.URL.Host
	}
	signature, err := (api.TC3Signer{}).Sign(auth.New(demoCredentials.AccessKey, demoCredentials.SecretKey, ""), api.SignInput{
		Method:      req.Method,
		Service:     service,
		Host:        host,
		Path:        demoreplay.FirstNonEmpty(req.URL.Path, "/"),
		Query:       req.URL.RawQuery,
		ContentType: demoreplay.FirstNonEmpty(req.Header.Get("Content-Type"), "application/json"),
		Timestamp:   timestamp,
		Payload:     body,
	})
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	if !demoreplay.SubtleEqual(signature.Authorization, authHeader) {
		return demoreplay.AuthInvalidSignature
	}
	return demoreplay.AuthOK
}

func verifyCOSAuth(req *http.Request) demoreplay.AuthFailureKind {
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))
	values, err := parseCOSAuthorization(authHeader)
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	if !demoreplay.SubtleEqual(values["q-ak"], demoCredentials.AccessKey) {
		return demoreplay.AuthInvalidAccessKey
	}
	signRange := strings.TrimSpace(values["q-sign-time"])
	parts := strings.Split(signRange, ";")
	if len(parts) != 2 {
		return demoreplay.AuthInvalidSignature
	}
	startUnix, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	endUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || endUnix-startUnix != int64(time.Hour/time.Second) {
		return demoreplay.AuthInvalidSignature
	}

	clone := req.Clone(context.Background())
	clone.Header = req.Header.Clone()
	clone.Header.Del("Authorization")
	if err := cos.Sign(clone, auth.New(demoCredentials.AccessKey, demoCredentials.SecretKey, ""), time.Unix(startUnix, 0).UTC()); err != nil {
		return demoreplay.AuthInvalidSignature
	}
	if !demoreplay.SubtleEqual(strings.TrimSpace(clone.Header.Get("Authorization")), authHeader) {
		return demoreplay.AuthInvalidSignature
	}
	return demoreplay.AuthOK
}

func parseCOSAuthorization(authHeader string) (map[string]string, error) {
	authHeader = strings.TrimSpace(authHeader)
	items := make(map[string]string)
	for _, part := range strings.Split(authHeader, "&") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid cos authorization part: %s", part)
		}
		decodedKey, err := url.QueryUnescape(key)
		if err != nil {
			return nil, err
		}
		decodedValue, err := url.QueryUnescape(value)
		if err != nil {
			return nil, err
		}
		items[decodedKey] = decodedValue
	}
	return items, nil
}

func requestService(req *http.Request) string {
	host := strings.ToLower(req.URL.Hostname())
	if strings.HasSuffix(host, ".tencentcloudapi.com") {
		if prefix, _, ok := strings.Cut(host, "."); ok {
			return prefix
		}
	}
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))
	_, service, ok := parseTC3Credential(authHeader)
	if !ok {
		return ""
	}
	return service
}

func parseTC3Credential(authHeader string) (string, string, bool) {
	authHeader = strings.TrimSpace(authHeader)
	if !strings.HasPrefix(authHeader, "TC3-HMAC-SHA256 Credential=") {
		return "", "", false
	}
	credential := strings.TrimPrefix(authHeader, "TC3-HMAC-SHA256 Credential=")
	scope, _, _ := strings.Cut(credential, ",")
	parts := strings.Split(scope, "/")
	if len(parts) < 4 {
		return "", "", false
	}
	return parts[0], parts[2], true
}

func parseUnixHeader(value string) (time.Time, error) {
	sec, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, 0).UTC(), nil
}

func isCOSHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "service.cos.myqcloud.com" || strings.Contains(host, ".cos.")
}

func parseBucketHost(host string) (string, string, bool) {
	prefix, rest, ok := strings.Cut(host, ".cos.")
	if !ok {
		return "", "", false
	}
	region, _, ok := strings.Cut(rest, ".myqcloud.com")
	if !ok {
		return "", "", false
	}
	if prefix == "" || region == "" {
		return "", "", false
	}
	return prefix, region, true
}

func openAPIErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	requestID := "req-replay-auth"
	if !strings.HasPrefix(code, "AuthFailure.") {
		requestID = requestIDForAction(req)
	}
	return demoreplay.JSONResponse(req, statusCode, map[string]any{
		"Response": map[string]any{
			"Error": map[string]string{
				"Code":    code,
				"Message": message,
			},
			"RequestId": requestID,
		},
	})
}

func requestIDForAction(req *http.Request) string {
	if req == nil {
		return "req-replay"
	}
	action := tcAction(req)
	if action == "" {
		return "req-replay"
	}
	return "req-replay-" + sanitizeAction(action)
}

func sanitizeAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	var b strings.Builder
	for _, r := range action {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "action"
	}
	return s
}

func xmlErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	return demoreplay.XMLResponse(req, statusCode, cosErrorResponse{
		Code:      code,
		Message:   message,
		Resource:  req.URL.Path,
		RequestID: "req-replay-cos",
		TraceID:   "trace-replay-cos",
	})
}

type cosErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource"`
	RequestID string   `xml:"RequestId"`
	TraceID   string   `xml:"TraceId"`
}

func (t *transport) snapshotUsers() map[string]camUserFixture {
	t.mu.Lock()
	defer t.mu.Unlock()
	items := make(map[string]camUserFixture, len(t.createdUsers))
	for name, user := range t.createdUsers {
		items[name] = user
	}
	return items
}

func (t *transport) ensureUser(name, password string) camUserFixture {
	name = strings.TrimSpace(name)
	password = strings.TrimSpace(password)
	t.mu.Lock()
	defer t.mu.Unlock()
	if user, ok := t.createdUsers[name]; ok {
		return user
	}
	t.sequence++
	user := camUserFixture{
		UIN:          uint64(200000000 + t.sequence),
		Name:         name,
		ConsoleLogin: password != "",
		CreateTime:   "2026-04-23 23:00:00",
		Policies: []camPolicyFixture{
			demoPolicies[0],
		},
	}
	t.createdUsers[name] = user
	return user
}

func (t *transport) deleteUser(name string) {
	name = strings.TrimSpace(name)
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.createdUsers, name)
}

func (t *transport) findUserByName(name string) (camUserFixture, bool) {
	name = strings.TrimSpace(name)
	t.mu.Lock()
	defer t.mu.Unlock()
	if user, ok := t.createdUsers[name]; ok {
		return user, true
	}
	for _, user := range demoCAMUsers {
		if user.Name == name {
			return user, true
		}
	}
	return camUserFixture{}, false
}

func (t *transport) findUserByUIN(uin uint64) (camUserFixture, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, user := range t.createdUsers {
		if user.UIN == uin {
			return user, true
		}
	}
	for _, user := range demoCAMUsers {
		if user.UIN == uin {
			return user, true
		}
	}
	return camUserFixture{}, false
}

func (t *transport) newInvocation(instanceID, output string) invocationResult {
	instanceID = strings.TrimSpace(instanceID)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sequence++
	commandID := fmt.Sprintf("cmd-replay-%03d", t.sequence)
	invocationID := fmt.Sprintf("ivk-replay-%03d", t.sequence)
	taskID := fmt.Sprintf("task-replay-%03d", t.sequence)
	result := invocationResult{
		CommandID:    commandID,
		InvocationID: invocationID,
		TaskID:       taskID,
		InstanceID:   instanceID,
		Output:       output,
	}
	t.invocations[invocationID] = result
	t.tasks[taskID] = result
	return result
}

func (t *transport) findInvocation(invocationID string) (invocationResult, bool) {
	invocationID = strings.TrimSpace(invocationID)
	t.mu.Lock()
	defer t.mu.Unlock()
	result, ok := t.invocations[invocationID]
	return result, ok
}

func (t *transport) findTask(taskID string) (invocationResult, bool) {
	taskID = strings.TrimSpace(taskID)
	t.mu.Lock()
	defer t.mu.Unlock()
	result, ok := t.tasks[taskID]
	return result, ok
}

func decodeCommandContent(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return content
	}
	return string(decoded)
}

func tcAction(req *http.Request) string {
	if req == nil {
		return ""
	}
	return strings.TrimSpace(req.Header.Get("X-TC-Action"))
}

func tcRegion(req *http.Request) string {
	if req == nil {
		return ""
	}
	return strings.TrimSpace(req.Header.Get("X-TC-Region"))
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefUint64(v *uint64) uint64 {
	if v == nil {
		return 0
	}
	return *v
}

func derefInt64(v *int64, fallback ...int64) int64 {
	if v != nil {
		return *v
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return 0
}

func stringPtr(v string) *string {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}
