package iam

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/utils/logger"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

var policy_infos map[string]string

func listAttachedUserAllPolicies(client *cam.Client, uin *uint64) string {
	request := cam.NewListAttachedUserAllPoliciesRequest()
	request.TargetUin = uin
	request.Rp = common.Uint64Ptr(20)
	request.Page = common.Uint64Ptr(1)
	request.AttachType = common.Uint64Ptr(0)

	resp, err := client.ListAttachedUserAllPolicies(request)
	if err != nil {
		return ""
	}
	policies := []string{}
	for _, p := range resp.Response.PolicyList {
		policies = append(policies, *p.PolicyName)
		if *p.StrategyType == "1" {
			if _, ok := policy_infos[*p.PolicyName]; !ok {
				details := getPolicy(client, *p.PolicyId)
				policy_infos[*p.PolicyName] = details
				msg := fmt.Sprintf("Found Custom Policy %s: %s", *p.PolicyName, details)
				logger.Warning(msg)
			}
		}
	}
	return strings.Join(policies, "\n")
}

func getPolicy(client *cam.Client, pid string) string {
	request := cam.NewGetPolicyRequest()
	pid_int, _ := strconv.Atoi(pid)
	request.PolicyId = common.Uint64Ptr(uint64(pid_int))

	response, err := client.GetPolicy(request)
	if err != nil {
		return err.Error()
	}
	return *response.Response.PolicyDocument
}
