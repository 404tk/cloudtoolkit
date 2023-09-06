package ram

import (
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

var policy_infos map[string]string

func listPoliciesForUser(client *ram.Client, name string) string {
	req_perm := ram.CreateListPoliciesForUserRequest()
	req_perm.Scheme = "https"
	req_perm.UserName = name
	resp, err := client.ListPoliciesForUser(req_perm)
	if err != nil {
		return ""
	}
	policies := []string{}
	for _, p := range resp.Policies.Policy {
		policies = append(policies, p.PolicyName)
		if p.PolicyType == "Custom" {
			if _, ok := policy_infos[p.PolicyName]; !ok {
				details := getPolicy(client, p.PolicyName)
				policy_infos[p.PolicyName] = details
				msg := fmt.Sprintf("Found Custom Policy %s: %s", p.PolicyName, details)
				logger.Warning(msg)
			}
		}
	}
	return strings.Join(policies, "\n")
}

func getPolicy(client *ram.Client, name string) string {
	request := ram.CreateGetPolicyRequest()
	request.Scheme = "https"
	request.PolicyName = name
	request.PolicyType = "Custom"
	response, err := client.GetPolicy(request)
	if err != nil {
		return err.Error()
	}
	return response.DefaultPolicyVersion.PolicyDocument
}
