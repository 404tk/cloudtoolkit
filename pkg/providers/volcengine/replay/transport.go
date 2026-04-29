package replay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/tos"
)

type invocationResult struct {
	CommandID  string
	InstanceID string
	Output     string
}

type transport struct {
	mu          sync.Mutex
	sequence    int
	commands    map[string]string
	invocations map[string]invocationResult
}

func newTransport() *transport {
	return &transport{
		commands:    make(map[string]string),
		invocations: make(map[string]invocationResult),
	}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := demoreplay.ReadRequestBody(req)
	if err != nil {
		return nil, err
	}
	if isTOSHost(req.URL.Hostname()) {
		return t.handleTOS(req, body)
	}
	return t.handleOpenAPI(req, body)
}

func (t *transport) handleOpenAPI(req *http.Request, body []byte) (*http.Response, error) {
	switch verifyOpenAPIAuth(req, body) {
	case demoreplay.AuthInvalidAccessKey:
		return openAPIErrorResponse(req, http.StatusForbidden, "AuthFailure.InvalidAccessKey", "The specified access key is invalid."), nil
	case demoreplay.AuthInvalidSignature:
		return openAPIErrorResponse(req, http.StatusForbidden, "AuthFailure.SignatureDoesNotMatch", "The request signature we calculated does not match the signature you provided."), nil
	}

	service := openAPIService(req.URL.Hostname())
	action := strings.TrimSpace(req.URL.Query().Get("Action"))
	switch service {
	case "billing":
		return t.handleBilling(req, action)
	case "iam":
		return t.handleIAM(req, action, body)
	case "dns":
		return t.handleDNS(req, action, body)
	case "ecs":
		return t.handleECS(req, action)
	case api.ServiceRDSMySQL:
		return t.handleRDSMySQL(req, action, body)
	case api.ServiceRDSPostgreSQL:
		return t.handleRDSPostgreSQL(req, action, body)
	case api.ServiceRDSMSSQL:
		return t.handleRDSSQLServer(req, action, body)
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction", fmt.Sprintf("unsupported replay service: %s", service)), nil
	}
}

func (t *transport) handleBilling(req *http.Request, action string) (*http.Response, error) {
	if action != "QueryBalanceAcct" {
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction", "unsupported billing action"), nil
	}
	resp := api.QueryBalanceAcctResponse{}
	resp.ResponseMetadata.RequestID = "req-billing-balance"
	resp.Result.AvailableBalance = "1024.88"
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleIAM(req *http.Request, action string, body []byte) (*http.Response, error) {
	query := req.URL.Query()
	switch action {
	case "ListProjects":
		resp := api.ListProjectsResponse{}
		resp.ResponseMetadata.RequestID = "req-iam-projects"
		resp.Result.Projects = []api.IAMProject{{
			ProjectName: demoProject,
			AccountID:   demoAccountID,
		}}
		resp.Result.Total = 1
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "ListUsers":
		limit := parseInt32(query.Get("Limit"), 100)
		offset := parseInt32(query.Get("Offset"), 0)
		window := demoreplay.OffsetWindow(len(demoIAMUsers), int(offset), int(limit))
		resp := api.ListUsersResponse{}
		resp.ResponseMetadata.RequestID = "req-iam-list-users"
		resp.Result.Total = int32(len(demoIAMUsers))
		resp.Result.Limit = limit
		resp.Result.Offset = offset
		resp.Result.UserMetadata = make([]api.IAMUserMetadata, 0, window.End-window.Start)
		for _, user := range demoIAMUsers[window.Start:window.End] {
			resp.Result.UserMetadata = append(resp.Result.UserMetadata, api.IAMUserMetadata{
				UserName:   user.UserName,
				AccountID:  user.AccountID,
				CreateDate: user.CreateDate,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "GetLoginProfile":
		user, ok := findUser(query.Get("UserName"))
		if !ok || !user.LoginAllowed {
			return openAPIErrorResponse(req, http.StatusNotFound, "EntityNotExist.LoginProfile", "login profile not found"), nil
		}
		resp := api.GetLoginProfileResponse{}
		resp.ResponseMetadata.RequestID = "req-iam-login-profile"
		resp.Result.LoginProfile = api.IAMLoginProfile{
			UserName:              user.UserName,
			LastLoginDate:         user.LastLoginDate,
			LoginAllowed:          user.LoginAllowed,
			PasswordResetRequired: false,
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "CreateUser":
		userName := strings.TrimSpace(query.Get("UserName"))
		resp := api.CreateUserResponse{}
		resp.ResponseMetadata.RequestID = "req-iam-create-user"
		resp.Result.User = api.IAMUserMetadata{
			UserName:   userName,
			AccountID:  demoAccountID,
			CreateDate: "20260422T120000Z",
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "CreateLoginProfile":
		var payload struct {
			UserName              string `json:"UserName"`
			LoginAllowed          bool   `json:"LoginAllowed"`
			PasswordResetRequired bool   `json:"PasswordResetRequired"`
		}
		_ = json.Unmarshal(body, &payload)
		resp := api.CreateLoginProfileResponse{}
		resp.ResponseMetadata.RequestID = "req-iam-create-login-profile"
		resp.Result.LoginProfile = api.IAMLoginProfile{
			UserName:              strings.TrimSpace(payload.UserName),
			LoginAllowed:          true,
			PasswordResetRequired: false,
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "AttachUserPolicy":
		resp := api.AttachUserPolicyResponse{}
		resp.ResponseMetadata.RequestID = "req-iam-attach-policy"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DetachUserPolicy":
		resp := api.DetachUserPolicyResponse{}
		resp.ResponseMetadata.RequestID = "req-iam-detach-policy"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DeleteLoginProfile":
		resp := api.DeleteLoginProfileResponse{}
		resp.ResponseMetadata.RequestID = "req-iam-delete-login-profile"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DeleteUser":
		resp := api.DeleteUserResponse{}
		resp.ResponseMetadata.RequestID = "req-iam-delete-user"
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction", fmt.Sprintf("unsupported iam action: %s", action)), nil
	}
}

func (t *transport) handleDNS(req *http.Request, action string, body []byte) (*http.Response, error) {
	switch action {
	case "ListZones":
		resp := api.ListDNSZonesResponse{
			Total: int32(len(demoZones)),
			Zones: make([]api.DNSZone, 0, len(demoZones)),
		}
		for _, zone := range demoZones {
			resp.Zones = append(resp.Zones, api.DNSZone{
				ZID:      zone.ID,
				ZoneName: zone.Name,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "ListRecords":
		var payload struct {
			ZID int64 `json:"ZID"`
		}
		_ = json.Unmarshal(body, &payload)
		zone, ok := findZone(payload.ZID)
		if !ok {
			return demoreplay.JSONResponse(req, http.StatusOK, api.ListDNSRecordsResponse{}), nil
		}
		resp := api.ListDNSRecordsResponse{
			PageNumber: 1,
			PageSize:   100,
			Records:    make([]api.DNSRecord, 0, len(zone.Records)),
			TotalCount: int32(len(zone.Records)),
		}
		for _, record := range zone.Records {
			enable := record.Enable
			resp.Records = append(resp.Records, api.DNSRecord{
				Enable: &enable,
				FQDN:   fqdnForRecord(zone.Name, record.Host),
				Host:   record.Host,
				Type:   record.Type,
				Value:  record.Value,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction", fmt.Sprintf("unsupported dns action: %s", action)), nil
	}
}

func (t *transport) handleECS(req *http.Request, action string) (*http.Response, error) {
	query := req.URL.Query()
	switch action {
	case "DescribeRegions":
		resp := api.DescribeRegionsResponse{}
		resp.ResponseMetadata.RequestID = "req-ecs-regions"
		for _, region := range demoRegions() {
			resp.Result.Regions = append(resp.Result.Regions, api.ECSRegion{RegionID: region})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeInstances":
		region := requestRegion(req)
		resp := api.DescribeInstancesResponse{}
		resp.ResponseMetadata.RequestID = "req-ecs-instances"
		for _, host := range hostsForRegion(region) {
			resp.Result.Instances = append(resp.Result.Instances, api.ECSInstance{
				InstanceID: host.InstanceID,
				Hostname:   host.Hostname,
				Status:     host.Status,
				OSType:     host.OSType,
				EipAddress: api.ECSEipAddress{
					IPAddress: host.PublicIP,
				},
				NetworkInterfaces: []api.ECSNetworkInterface{{
					PrimaryIPAddress: host.PrivateIP,
				}},
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeCloudAssistantStatus":
		instanceID := strings.TrimSpace(demoreplay.FirstNonEmpty(query.Get("InstanceIds.1"), query.Get("InstanceId")))
		host, ok := findHost(instanceID)
		if !ok {
			return openAPIErrorResponse(req, http.StatusNotFound, "InvalidInstance.NotFound", "the specified instance does not exist"), nil
		}
		resp := api.DescribeCloudAssistantStatusResponse{}
		resp.ResponseMetadata.RequestID = "req-ecs-cloud-assistant"
		resp.Result.PageNumber = 1
		resp.Result.PageSize = 20
		resp.Result.TotalCount = 1
		resp.Result.Instances = []api.ECSCloudAssistantInstance{{
			InstanceID:        host.InstanceID,
			HostName:          host.Hostname,
			InstanceName:      host.Hostname,
			Status:            host.AgentStatus,
			ClientVersion:     "1.0.0",
			OSType:            host.OSType,
			OSVersion:         "Demo Linux",
			LastHeartbeatTime: "2026-04-22T12:00:00Z",
		}}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "CreateCommand":
		command := strings.TrimSpace(query.Get("CommandContent"))
		if strings.EqualFold(query.Get("ContentEncoding"), "Base64") {
			if decoded, err := base64.StdEncoding.DecodeString(command); err == nil {
				command = string(decoded)
			}
		}
		commandID := t.nextID("cmd")
		t.mu.Lock()
		t.commands[commandID] = command
		t.mu.Unlock()
		resp := api.CreateCommandResponse{}
		resp.ResponseMetadata.RequestID = "req-ecs-create-command"
		resp.Result.CommandID = commandID
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "InvokeCommand":
		commandID := strings.TrimSpace(query.Get("CommandId"))
		instanceID := strings.TrimSpace(query.Get("InstanceIds.1"))
		t.mu.Lock()
		command := t.commands[commandID]
		invocationID := t.nextIDLocked("ivk")
		t.invocations[invocationID] = invocationResult{
			CommandID:  commandID,
			InstanceID: instanceID,
			Output:     base64.StdEncoding.EncodeToString([]byte(shellOutput(instanceID, command))),
		}
		t.mu.Unlock()
		resp := api.InvokeCommandResponse{}
		resp.ResponseMetadata.RequestID = "req-ecs-invoke-command"
		resp.Result.InvocationID = invocationID
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeInvocationResults":
		invocationID := strings.TrimSpace(query.Get("InvocationId"))
		t.mu.Lock()
		result, ok := t.invocations[invocationID]
		t.mu.Unlock()
		if !ok {
			return openAPIErrorResponse(req, http.StatusNotFound, "InvalidInvocation.NotFound", "the specified invocation does not exist"), nil
		}
		resp := api.DescribeInvocationResultsResponse{}
		resp.ResponseMetadata.RequestID = "req-ecs-invocation-results"
		resp.Result.PageNumber = 1
		resp.Result.PageSize = 1
		resp.Result.TotalCount = 1
		resp.Result.InvocationResults = []api.ECSInvocationResult{{
			CommandID:              result.CommandID,
			InvocationID:           invocationID,
			InstanceID:             result.InstanceID,
			InvocationResultStatus: "Success",
			Output:                 result.Output,
		}}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DeleteCommand":
		commandID := strings.TrimSpace(query.Get("CommandId"))
		t.mu.Lock()
		delete(t.commands, commandID)
		t.mu.Unlock()
		resp := api.DeleteCommandResponse{}
		resp.ResponseMetadata.RequestID = "req-ecs-delete-command"
		resp.Result.CommandID = commandID
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	default:
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction", fmt.Sprintf("unsupported ecs action: %s", action)), nil
	}
}

func (t *transport) handleRDSMySQL(req *http.Request, action string, body []byte) (*http.Response, error) {
	return t.handleRDS(req, action, body, api.ServiceRDSMySQL)
}

func (t *transport) handleRDSPostgreSQL(req *http.Request, action string, body []byte) (*http.Response, error) {
	return t.handleRDS(req, action, body, api.ServiceRDSPostgreSQL)
}

func (t *transport) handleRDSSQLServer(req *http.Request, action string, body []byte) (*http.Response, error) {
	return t.handleRDS(req, action, body, api.ServiceRDSMSSQL)
}

func (t *transport) handleRDS(req *http.Request, action string, body []byte, service string) (*http.Response, error) {
	switch action {
	case "DescribeRegions":
		resp := api.DescribeRDSRegionsResponse{}
		resp.ResponseMetadata.RequestID = "req-rds-regions-" + service
		for _, region := range demoRegions() {
			resp.Result.Regions = append(resp.Result.Regions, api.RDSRegion{
				RegionID:   region,
				RegionName: region,
			})
		}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeDBInstances":
		var payload struct {
			PageNumber int32 `json:"PageNumber"`
			PageSize   int32 `json:"PageSize"`
		}
		_ = json.Unmarshal(body, &payload)
		region := requestRegion(req)
		switch service {
		case api.ServiceRDSMySQL:
			resp := api.DescribeRDSMySQLInstancesResponse{}
			resp.ResponseMetadata.RequestID = "req-rds-mysql-instances"
			for _, item := range mysqlForRegion(region) {
				resp.Result.Instances = append(resp.Result.Instances, api.RDSMySQLInstance{
					InstanceID:      item.InstanceID,
					DBEngineVersion: item.Version,
					RegionID:        item.Region,
					AddressObject: []api.RDSAddressObject{
						{NetworkType: "Private", IPAddress: item.PrivateIP, Port: item.Port},
						{NetworkType: "Public", Domain: item.PublicHost, Port: item.Port},
					},
				})
			}
			resp.Result.Total = int32(len(resp.Result.Instances))
			return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
		case api.ServiceRDSPostgreSQL:
			resp := api.DescribeRDSPostgreSQLInstancesResponse{}
			resp.ResponseMetadata.RequestID = "req-rds-postgresql-instances"
			for _, item := range postgresForRegion(region) {
				resp.Result.Instances = append(resp.Result.Instances, api.RDSPostgreSQLInstance{
					InstanceID:      item.InstanceID,
					DBEngineVersion: item.Version,
					RegionID:        item.Region,
					AddressObject: []api.RDSAddressObject{
						{NetworkType: "Private", IPAddress: item.PrivateIP, Port: item.Port},
					},
				})
			}
			resp.Result.Total = int32(len(resp.Result.Instances))
			return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
		case api.ServiceRDSMSSQL:
			resp := api.DescribeRDSSQLServerInstancesResponse{}
			resp.ResponseMetadata.RequestID = "req-rds-sqlserver-instances"
			for _, item := range sqlServerForRegion(region) {
				resp.Result.InstancesInfo = append(resp.Result.InstancesInfo, api.RDSSQLServerInstance{
					InstanceID:      item.InstanceID,
					DBEngineVersion: item.Version,
					RegionID:        item.Region,
					Port:            item.Port,
					NodeDetailInfo: []api.RDSSQLServerNode{{
						NodeType: "Primary",
						NodeIP:   item.PrimaryIP,
					}},
				})
			}
			resp.Result.Total = int32(len(resp.Result.InstancesInfo))
			return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
		}
	}
	return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction", fmt.Sprintf("unsupported rds action: %s", action)), nil
}

func (t *transport) handleTOS(req *http.Request, body []byte) (*http.Response, error) {
	switch verifyTOSAuth(req, body) {
	case demoreplay.AuthInvalidAccessKey:
		return tosErrorResponse(req, http.StatusForbidden, "InvalidAccessKey", "The access key is invalid."), nil
	case demoreplay.AuthInvalidSignature:
		return tosErrorResponse(req, http.StatusForbidden, "SignatureDoesNotMatch", "The request signature we calculated does not match the signature you provided."), nil
	}

	host := strings.ToLower(strings.TrimSpace(req.URL.Hostname()))
	query := req.URL.Query()
	if bucket, region, ok := parseBucketHost(host); ok {
		if query.Get("list-type") == "2" {
			return t.handleListObjects(req, bucket, region, query)
		}
		return tosErrorResponse(req, http.StatusNotFound, "InvalidRequest", "unsupported tos request"), nil
	}
	if isServiceHost(host) {
		return t.handleListBuckets(req)
	}
	return tosErrorResponse(req, http.StatusNotFound, "InvalidRequest", "unsupported tos host"), nil
}

func (t *transport) handleListBuckets(req *http.Request) (*http.Response, error) {
	items := demoBuckets
	resp := tos.ListBucketsOutput{
		Buckets: make([]tos.Bucket, 0, len(items)),
	}
	resp.Owner.ID = strconv.FormatInt(demoAccountID, 10)
	for _, bucket := range items {
		resp.Buckets = append(resp.Buckets, tos.Bucket{
			Name:             bucket.Name,
			Location:         bucket.Region,
			CreationDate:     "2026-04-22T12:00:00.000Z",
			ExtranetEndpoint: "tos-" + bucket.Region + ".volces.com",
			IntranetEndpoint: "tos-" + bucket.Region + ".ivolces.com",
			ProjectName:      demoProject,
			BucketType:       "Standard",
		})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleListObjects(req *http.Request, bucketName, region string, query url.Values) (*http.Response, error) {
	bucket, ok := findBucket(bucketName)
	if !ok {
		return tosErrorResponse(req, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist."), nil
	}
	if region != "" && bucket.Region != region {
		return tosErrorResponse(req, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist in this region."), nil
	}

	maxKeys := demoreplay.ParseInt(query.Get("max-keys"), 1000)
	offset := demoreplay.ParseInt(query.Get("continuation-token"), 0)
	window := demoreplay.OffsetWindow(len(bucket.Objects), offset, maxKeys)

	resp := tos.ListObjectsV2Output{
		Name:              bucket.Name,
		MaxKeys:           maxKeys,
		ContinuationToken: query.Get("continuation-token"),
		IsTruncated:       window.End < len(bucket.Objects),
		Contents:          make([]tos.BucketItem, 0, window.End-window.Start),
	}
	if resp.IsTruncated {
		resp.NextContinuationToken = strconv.Itoa(window.End)
	}
	for _, item := range bucket.Objects[window.Start:window.End] {
		resp.Contents = append(resp.Contents, tos.BucketItem{
			Key:          item.Key,
			Size:         item.Size,
			LastModified: "2026-04-22T12:00:00.000Z",
			StorageClass: "STANDARD",
			Type:         "Normal",
		})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func verifyOpenAPIAuth(req *http.Request, body []byte) demoreplay.AuthFailureKind {
	parsed, ok := parseAuthorization(req.Header.Get(api.HeaderAuthorization))
	if !ok {
		return demoreplay.AuthInvalidSignature
	}
	if parsed.AccessKey != demoCredentials.AccessKey {
		return demoreplay.AuthInvalidAccessKey
	}
	xDate := strings.TrimSpace(req.Header.Get(api.HeaderXDate))
	timestamp, err := time.Parse(api.DateFormat, xDate)
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	host := normalizeHost(req.Host)
	if host == "" {
		host = normalizeHost(req.URL.Host)
	}
	contentType := strings.TrimSpace(req.Header.Get("Content-Type"))
	sessionToken := strings.TrimSpace(req.Header.Get(api.HeaderXSecurityToken))
	expected, err := api.Sign(api.SignInput{
		Method:       req.Method,
		Host:         host,
		Path:         req.URL.Path,
		Query:        httpclient.CloneValues(req.URL.Query()),
		Body:         body,
		ContentType:  contentType,
		Service:      parsed.Service,
		Region:       parsed.Region,
		AccessKey:    demoCredentials.AccessKey,
		SecretKey:    demoCredentials.SecretKey,
		SessionToken: sessionToken,
		Headers:      req.Header.Clone(),
		Timestamp:    timestamp,
	})
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	if demoreplay.SubtleEqual(strings.TrimSpace(expected.Authorization), strings.TrimSpace(req.Header.Get(api.HeaderAuthorization))) {
		return demoreplay.AuthOK
	}
	return demoreplay.AuthInvalidSignature
}

func verifyTOSAuth(req *http.Request, body []byte) demoreplay.AuthFailureKind {
	parsed, ok := parseAuthorization(req.Header.Get("Authorization"))
	if !ok {
		return demoreplay.AuthInvalidSignature
	}
	if parsed.AccessKey != demoCredentials.AccessKey {
		return demoreplay.AuthInvalidAccessKey
	}
	xTosDate := strings.TrimSpace(req.Header.Get("X-Tos-Date"))
	timestamp, err := time.Parse("20060102T150405Z", xTosDate)
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	expected, err := signTOSRequest(req, body, timestamp)
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	if demoreplay.SubtleEqual(strings.TrimSpace(expected), strings.TrimSpace(req.Header.Get("Authorization"))) {
		return demoreplay.AuthOK
	}
	return demoreplay.AuthInvalidSignature
}

type authorization struct {
	AccessKey string
	Region    string
	Service   string
}

func parseAuthorization(value string) (authorization, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return authorization{}, false
	}
	parts := strings.Split(value, ",")
	if len(parts) < 3 {
		return authorization{}, false
	}
	first := strings.Fields(parts[0])
	if len(first) != 2 {
		return authorization{}, false
	}
	credentialPart := strings.TrimPrefix(first[1], "Credential=")
	scope := strings.Split(credentialPart, "/")
	if len(scope) < 5 {
		return authorization{}, false
	}
	accessKey := strings.TrimSpace(scope[0])
	region := strings.TrimSpace(scope[2])
	service := strings.TrimSpace(scope[3])
	return authorization{
		AccessKey: accessKey,
		Region:    region,
		Service:   service,
	}, true
}

func signTOSRequest(req *http.Request, body []byte, timestamp time.Time) (string, error) {
	host := normalizeHost(req.Host)
	if host == "" {
		host = normalizeHost(req.URL.Host)
	}
	region := regionFromTOSHost(host)
	if region == "" {
		region = api.DefaultRegion
	}
	xDate := timestamp.UTC().Format("20060102T150405Z")
	shortDate := xDate[:8]
	payloadHash := sha256Hex(body)

	headers := canonicalTOSHeaders(req.Header.Clone(), host, payloadHash, xDate)
	signedHeaders := signedTOSHeaderNames(headers)
	canonicalRequest := strings.Join([]string{
		strings.ToUpper(req.Method),
		canonicalURI(req.URL.Path),
		canonicalQuery(req.URL.Query()),
		canonicalTOSHeadersText(headers, signedHeaders),
		strings.Join(signedHeaders, ";"),
		payloadHash,
	}, "\n")
	credentialScope := shortDate + "/" + region + "/tos/request"
	stringToSign := strings.Join([]string{
		"TOS4-HMAC-SHA256",
		xDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signature := hex.EncodeToString(signTOS(demoCredentials.SecretKey, shortDate, region, stringToSign))
	return fmt.Sprintf(
		"TOS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		demoCredentials.AccessKey,
		credentialScope,
		strings.Join(signedHeaders, ";"),
		signature,
	), nil
}

func canonicalTOSHeaders(headers http.Header, host, payloadHash, xDate string) map[string]string {
	items := map[string]string{
		"host":                 host,
		"x-tos-content-sha256": payloadHash,
		"x-tos-date":           xDate,
	}
	if token := strings.TrimSpace(headers.Get("X-Tos-Security-Token")); token != "" {
		items["x-tos-security-token"] = token
	}
	return items
}

func signedTOSHeaderNames(headers map[string]string) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		if name == "host" || strings.HasPrefix(name, "x-tos-") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func canonicalTOSHeadersText(headers map[string]string, signedHeaders []string) string {
	lines := make([]string, 0, len(signedHeaders))
	for _, name := range signedHeaders {
		lines = append(lines, name+":"+strings.TrimSpace(headers[name]))
	}
	return strings.Join(lines, "\n") + "\n"
}

func signTOS(secretKey, shortDate, region, stringToSign string) []byte {
	dateKey := hmacSHA256([]byte(secretKey), shortDate)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, "tos")
	signingKey := hmacSHA256(serviceKey, "request")
	return hmacSHA256(signingKey, stringToSign)
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func openAPIErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	code = strings.TrimSpace(code)
	message = strings.TrimSpace(message)
	payload := map[string]any{
		"ResponseMetadata": map[string]any{
			"RequestId": "req-error",
			"Error": map[string]any{
				"Code":    code,
				"Message": message,
			},
		},
	}
	return demoreplay.JSONResponse(req, statusCode, payload)
}

func tosErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	code = strings.TrimSpace(code)
	message = strings.TrimSpace(message)
	payload := map[string]any{
		"Code":      code,
		"Message":   message,
		"RequestId": "req-tos-error",
	}
	return demoreplay.JSONResponse(req, statusCode, payload)
}

func openAPIService(host string) string {
	host = normalizeHost(host)
	switch {
	case strings.HasPrefix(host, "billing."):
		return "billing"
	case strings.HasPrefix(host, "iam."):
		return "iam"
	case strings.HasPrefix(host, "dns."):
		return "dns"
	case strings.HasPrefix(host, "ecs."):
		return "ecs"
	case strings.HasPrefix(host, "rds-mysql."):
		return api.ServiceRDSMySQL
	case strings.HasPrefix(host, "rds-postgresql."):
		return api.ServiceRDSPostgreSQL
	case strings.HasPrefix(host, "rds-mssql."):
		return api.ServiceRDSMSSQL
	default:
		return ""
	}
}

func requestRegion(req *http.Request) string {
	if parsed, ok := parseAuthorization(req.Header.Get(api.HeaderAuthorization)); ok && parsed.Region != "" {
		return parsed.Region
	}
	host := normalizeHost(req.URL.Hostname())
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		region := strings.TrimSpace(parts[1])
		if region != "" {
			return region
		}
	}
	return api.DefaultRegion
}

func parseBucketHost(host string) (string, string, bool) {
	host = normalizeHost(host)
	idx := strings.Index(host, ".tos-")
	if idx <= 0 {
		return "", "", false
	}
	bucket := host[:idx]
	rest := host[idx+1:]
	end := strings.Index(rest, ".")
	if end <= len("tos-") {
		return "", "", false
	}
	return bucket, strings.TrimPrefix(rest[:end], "tos-"), true
}

func isTOSHost(host string) bool {
	host = normalizeHost(host)
	return strings.Contains(host, ".volces.com") && strings.Contains(host, "tos-")
}

func isServiceHost(host string) bool {
	host = normalizeHost(host)
	return strings.HasPrefix(host, "tos-")
}

func serviceRegionFromHost(host string) string {
	host = normalizeHost(host)
	host = strings.TrimPrefix(host, "tos-")
	if idx := strings.Index(host, "."); idx > 0 {
		return host[:idx]
	}
	return ""
}

func regionFromTOSHost(host string) string {
	if bucket, region, ok := parseBucketHost(host); ok && bucket != "" {
		return region
	}
	return serviceRegionFromHost(host)
}

func fqdnForRecord(zoneName, host string) string {
	host = strings.TrimSpace(host)
	zoneName = strings.TrimSpace(zoneName)
	if host == "" || host == "@" {
		return zoneName
	}
	return host + "." + zoneName
}

func parseInt32(value string, fallback int32) int32 {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	got, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(got)
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.Contains(host, "://") {
		if parsed, err := url.Parse(host); err == nil && parsed.Host != "" {
			host = parsed.Host
		}
	}
	host = strings.TrimSuffix(host, ":443")
	host = strings.TrimSuffix(host, ":80")
	return strings.ToLower(host)
}

func canonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	return strings.ReplaceAll(values.Encode(), "+", "%20")
}

func canonicalURI(path string) string {
	path = httpclient.EnsureLeadingSlash(path)
	if path == "/" {
		return "/"
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = percentEncodeRFC3986(part)
	}
	return strings.Join(parts, "/")
}

func percentEncodeRFC3986(value string) string {
	if value == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(value) * 3)
	for i := 0; i < len(value); i++ {
		c := value[i]
		if isUnreserved(c) {
			builder.WriteByte(c)
			continue
		}
		builder.WriteByte('%')
		builder.WriteByte(upperhex[c>>4])
		builder.WriteByte(upperhex[c&15])
	}
	return builder.String()
}

func isUnreserved(c byte) bool {
	return ('A' <= c && c <= 'Z') ||
		('a' <= c && c <= 'z') ||
		('0' <= c && c <= '9') ||
		c == '-' || c == '_' || c == '.' || c == '~'
}

var upperhex = "0123456789ABCDEF"

func (t *transport) nextID(prefix string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.nextIDLocked(prefix)
}

func (t *transport) nextIDLocked(prefix string) string {
	t.sequence++
	return fmt.Sprintf("%s-%06d", prefix, t.sequence)
}
