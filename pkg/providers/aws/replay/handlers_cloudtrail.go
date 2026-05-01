package replay

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// demoCloudTrailEvents seeds a small set of canned management-plane events
// so the replay surface is exercisable end-to-end. Mirrors the alibaba SAS
// / tencent CloudAudit fixture style.
var demoCloudTrailEvents = []api.CloudTrailEvent{
	{
		EventID:     "ct-replay-iam-001",
		EventName:   "CreateUser",
		EventTime:   1714694400,
		EventSource: "iam.amazonaws.com",
		Username:    "ctk-demo-admin",
		AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
		Resources: []api.CloudTrailResource{
			{ResourceType: "AWS::IAM::User", ResourceName: "ctk-demo-bot"},
		},
	},
	{
		EventID:     "ct-replay-iam-002",
		EventName:   "AttachUserPolicy",
		EventTime:   1714694430,
		EventSource: "iam.amazonaws.com",
		Username:    "ctk-demo-admin",
		AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
		Resources: []api.CloudTrailResource{
			{ResourceType: "AWS::IAM::Policy", ResourceName: "AdministratorAccess"},
		},
	},
	{
		EventID:     "ct-replay-ec2-001",
		EventName:   "RunInstances",
		EventTime:   1714694460,
		EventSource: "ec2.amazonaws.com",
		Username:    "ctk-demo-admin",
		AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
		Resources: []api.CloudTrailResource{
			{ResourceType: "AWS::EC2::Instance", ResourceName: "i-0a1b2c3d4e5f60099"},
		},
	},
}

func (t *transport) handleCloudTrail(req *http.Request, body []byte) (*http.Response, error) {
	if req.Method != http.MethodPost {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "InvalidAction", "cloudtrail replay expects POST"), nil
	}
	target := strings.TrimSpace(req.Header.Get("X-Amz-Target"))
	if !strings.HasSuffix(target, ".LookupEvents") {
		return apiErrorResponse(req, http.StatusBadRequest, "InvalidAction", fmt.Sprintf("unsupported cloudtrail target: %s", target)), nil
	}
	resp := api.LookupEventsOutput{
		Events: append([]api.CloudTrailEvent(nil), demoCloudTrailEvents...),
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}
