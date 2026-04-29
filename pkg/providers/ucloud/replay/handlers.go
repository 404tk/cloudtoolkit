package replay

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
)

func (t *transport) handleGetUserInfo(req *http.Request) (*http.Response, error) {
	resp := api.GetUserInfoResponse{
		BaseResponse: newBase("GetUserInfoResponse"),
		DataSet: []api.UserInfo{{
			UserEmail: demoUserEmail,
			UserID:    demoUserID,
			UserName:  demoUserName,
		}},
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleGetProjectList(req *http.Request) (*http.Response, error) {
	resp := api.GetProjectListResponse{
		BaseResponse: newBase("GetProjectListResponse"),
		ProjectSet: []api.ProjectListInfo{{
			IsDefault:   true,
			ProjectID:   demoProjectID,
			ProjectName: demoProjectName,
		}},
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleGetRegion(req *http.Request) (*http.Response, error) {
	resp := api.GetRegionResponse{
		BaseResponse: newBase("GetRegionResponse"),
	}
	for _, region := range demoRegionList {
		resp.Regions = append(resp.Regions, api.RegionInfo{Region: region})
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleGetBalance(req *http.Request) (*http.Response, error) {
	resp := api.GetBalanceResponse{
		BaseResponse: newBase("GetBalanceResponse"),
		AccountInfo: api.AccountInfo{
			Amount:          "10240.00",
			AmountAvailable: "10240.00",
		},
	}
	return successResponse(req, resp), nil
}

func paginationOffsetLimit(params map[string]string, defaultLimit int) (int, int) {
	offset, _ := strconv.Atoi(strings.TrimSpace(params["Offset"]))
	if offset < 0 {
		offset = 0
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(params["Limit"]))
	if limit <= 0 {
		limit = defaultLimit
	}
	return offset, limit
}

func (t *transport) handleDescribeUHostInstance(req *http.Request, params map[string]string) (*http.Response, error) {
	region := strings.TrimSpace(params["Region"])
	all := uhostsForRegion(region)
	offset, limit := paginationOffsetLimit(params, 100)
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	if offset > len(all) {
		offset = len(all)
	}
	page := all[offset:end]

	resp := api.DescribeUHostInstanceResponse{
		BaseResponse: newBase("DescribeUHostInstanceResponse"),
		TotalCount:   len(all),
	}
	for _, host := range page {
		entry := api.UHostSet{
			Name:    host.Name,
			OsType:  host.OsType,
			State:   host.State,
			UHostID: host.UHostID,
		}
		if host.PrivateIP != "" {
			entry.IPSet = append(entry.IPSet, api.UHostIPSet{
				Default: "true",
				IP:      host.PrivateIP,
				IPMode:  "IPv4",
				Type:    "Private",
			})
		}
		if host.PublicIP != "" {
			entry.IPSet = append(entry.IPSet, api.UHostIPSet{
				Default: "true",
				IP:      host.PublicIP,
				IPMode:  "IPv4",
				Type:    "International",
				Weight:  100,
			})
		}
		resp.UHostSet = append(resp.UHostSet, entry)
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleDescribeBucket(req *http.Request, params map[string]string) (*http.Response, error) {
	region := strings.TrimSpace(params["Region"])
	all := bucketsForRegion(region)
	offset, limit := paginationOffsetLimit(params, 100)
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	if offset > len(all) {
		offset = len(all)
	}
	page := all[offset:end]

	resp := api.DescribeBucketResponse{
		BaseResponse: newBase("DescribeBucketResponse"),
	}
	for _, bucket := range page {
		resp.DataSet = append(resp.DataSet, api.UFileBucketSet{
			BucketName: bucket.BucketName,
			Region:     bucket.Region,
		})
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleDescribeUDBInstance(req *http.Request, params map[string]string) (*http.Response, error) {
	region := strings.TrimSpace(params["Region"])
	classType := strings.TrimSpace(params["ClassType"])
	all := udbForRegionAndClass(region, classType)
	offset, limit := paginationOffsetLimit(params, 100)
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	if offset > len(all) {
		offset = len(all)
	}
	page := all[offset:end]

	resp := api.DescribeUDBInstanceResponse{
		BaseResponse: newBase("DescribeUDBInstanceResponse"),
		TotalCount:   len(all),
	}
	for _, db := range page {
		resp.DataSet = append(resp.DataSet, api.UDBInstanceSet{
			DBID:         db.DBID,
			DBSubVersion: db.DBSubVersion,
			DBTypeID:     db.DBTypeID,
			Name:         db.Name,
			Port:         db.Port,
			VirtualIP:    db.VirtualIP,
		})
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleDescribeUDNSZone(req *http.Request, params map[string]string) (*http.Response, error) {
	region := strings.TrimSpace(params["Region"])
	zones := dnsZonesForRegion(region)
	offset, limit := paginationOffsetLimit(params, 100)
	end := offset + limit
	if end > len(zones) {
		end = len(zones)
	}
	if offset > len(zones) {
		offset = len(zones)
	}
	page := zones[offset:end]

	resp := api.DescribeUDNSZoneResponse{
		BaseResponse: newBase("DescribeUDNSZoneResponse"),
		TotalCount:   len(zones),
	}
	for _, zone := range page {
		resp.DNSZoneInfos = append(resp.DNSZoneInfos, api.ZoneInfo{
			DNSZoneID:   zone.DNSZoneID,
			DNSZoneName: zone.DNSZoneName,
		})
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleDescribeUDNSRecord(req *http.Request, params map[string]string) (*http.Response, error) {
	zoneID := strings.TrimSpace(params["DNSZoneId"])
	zone, ok := findDNSZone(zoneID)
	resp := api.DescribeUDNSRecordResponse{
		BaseResponse: newBase("DescribeUDNSRecordResponse"),
	}
	if !ok {
		return successResponse(req, resp), nil
	}
	resp.TotalCount = len(zone.Records)
	for _, record := range zone.Records {
		resp.RecordInfos = append(resp.RecordInfos, api.RecordInfo{
			Name: record.Name,
			Type: record.Type,
			ValueSet: []api.ValueSet{{
				Data:      record.Value,
				IsEnabled: record.IsEnabled,
			}},
		})
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleListUsers(req *http.Request) (*http.Response, error) {
	users := t.iam.snapshotUsers()
	resp := api.IAMListUsersResponse{
		BaseResponse: newBase("ListUsersResponse"),
		TotalCount:   len(users),
	}
	for _, user := range users {
		resp.Users = append(resp.Users, api.IAMUserSummary{
			CreatedAt:   user.CreatedAt,
			DisplayName: user.DisplayName,
			Email:       user.Email,
			Status:      user.Status,
			UserName:    user.UserName,
		})
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleCreateUser(req *http.Request, params map[string]string) (*http.Response, error) {
	name := strings.TrimSpace(params["UserName"])
	if name == "" {
		return errorResponse(req, http.StatusBadRequest, 400, "UserName is required"), nil
	}
	display := strings.TrimSpace(params["DisplayName"])
	if display == "" {
		display = name
	}
	user := t.iam.ensureUser(name)
	user.DisplayName = display
	resp := api.IAMCreateUserResponse{
		BaseResponse:    newBase("CreateUserResponse"),
		APIAccess:       false,
		AccessKeyID:     "",
		AccessKeySecret: "",
		CompanyID:       demoCompanyID,
		ConsoleAccess:   true,
		DisplayName:     display,
		Password:        strings.TrimSpace(params["Password"]),
		UserName:        name,
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleDeleteUser(req *http.Request, params map[string]string) (*http.Response, error) {
	name := strings.TrimSpace(params["UserName"])
	if name == "" {
		return errorResponse(req, http.StatusBadRequest, 400, "UserName is required"), nil
	}
	if !t.iam.deleteUser(name) {
		return errorResponse(req, http.StatusNotFound, 404,
			"sub user "+name+" not found"), nil
	}
	resp := api.IAMDeleteUserResponse{BaseResponse: newBase("DeleteUserResponse")}
	resp.Message = "success"
	return successResponse(req, resp), nil
}

func (t *transport) handleAttachPolicies(req *http.Request, params map[string]string) (*http.Response, error) {
	user := strings.TrimSpace(params["UserName"])
	for key, value := range params {
		if !strings.HasPrefix(key, "PolicyURNs.") {
			continue
		}
		t.iam.attachPolicy(user, value)
	}
	resp := api.IAMAttachPoliciesToUserResponse{BaseResponse: newBase("AttachPoliciesToUserResponse")}
	return successResponse(req, resp), nil
}
