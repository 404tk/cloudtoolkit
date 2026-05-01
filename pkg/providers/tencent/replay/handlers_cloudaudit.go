package replay

import (
	"fmt"
	"net/http"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
)

// demoCloudAuditEvents seeds a small fixture set so the replay surface gives
// the validation flow a recognisable view of what CloudAudit's LookUpEvents
// returns in production. The values are chosen to mirror common CSPM-tracked
// API events (CreateUser, AttachUserPolicy, RunInstances).
var demoCloudAuditEvents = []api.CloudAuditEvent{
	{
		EventID:         stringPtr("evt-replay-cam-001"),
		EventName:       stringPtr("CreateUser"),
		EventNameCn:     stringPtr("创建子用户"),
		EventTime:       stringPtr("2026-04-22 09:10:11"),
		EventRegion:     stringPtr("ap-guangzhou"),
		Username:        stringPtr("ctk-demo-admin"),
		SourceIPAddress: stringPtr("203.0.113.10"),
		ResourceTypeCn:  stringPtr("访问管理"),
		ResourceName:    stringPtr("ctk-demo-bot"),
		Status:          uint64Ptr(0),
		SecretID:        stringPtr(demoCredentials.AccessKey),
		APIVersion:      stringPtr("2019-01-16"),
	},
	{
		EventID:         stringPtr("evt-replay-cam-002"),
		EventName:       stringPtr("AttachUserPolicy"),
		EventNameCn:     stringPtr("授权子用户策略"),
		EventTime:       stringPtr("2026-04-22 09:10:42"),
		EventRegion:     stringPtr("ap-guangzhou"),
		Username:        stringPtr("ctk-demo-admin"),
		SourceIPAddress: stringPtr("203.0.113.10"),
		ResourceTypeCn:  stringPtr("访问管理"),
		ResourceName:    stringPtr("AdministratorAccess"),
		Status:          uint64Ptr(0),
		SecretID:        stringPtr(demoCredentials.AccessKey),
		APIVersion:      stringPtr("2019-01-16"),
	},
	{
		EventID:         stringPtr("evt-replay-cvm-001"),
		EventName:       stringPtr("RunInstances"),
		EventNameCn:     stringPtr("创建实例"),
		EventTime:       stringPtr("2026-04-22 09:11:03"),
		EventRegion:     stringPtr("ap-shanghai"),
		Username:        stringPtr("ctk-demo-admin"),
		SourceIPAddress: stringPtr("203.0.113.10"),
		ResourceTypeCn:  stringPtr("云服务器"),
		ResourceName:    stringPtr("ins-cvm003"),
		Status:          uint64Ptr(0),
		SecretID:        stringPtr(demoCredentials.AccessKey),
		APIVersion:      stringPtr("2017-03-12"),
	},
}

func (t *transport) handleCloudAudit(req *http.Request, action string) (*http.Response, error) {
	switch action {
	case "LookUpEvents":
		listOver := true
		resp := api.LookUpEventsResponse{}
		resp.Response.RequestID = "req-replay-cloudaudit-lookup"
		resp.Response.ListOver = &listOver
		resp.Response.Events = append([]api.CloudAuditEvent(nil), demoCloudAuditEvents...)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound", fmt.Sprintf("Unsupported replay action: %s", action)), nil
}
