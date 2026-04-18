package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

var policyInfos map[string]string

func listPoliciesForUser(ctx context.Context, client *api.Client, region, name string) string {
	resp, err := client.ListRAMPoliciesForUser(ctx, region, name)
	if err != nil {
		return ""
	}
	policies := []string{}
	for _, p := range resp.Policies.Policy {
		policies = append(policies, p.PolicyName)
		if p.PolicyType == "Custom" {
			if _, ok := policyInfos[p.PolicyName]; !ok {
				details := getPolicy(ctx, client, region, p.PolicyName)
				policyInfos[p.PolicyName] = details
				msg := fmt.Sprintf("Found Custom Policy %s: %s", p.PolicyName, details)
				logger.Warning(msg)
			}
		}
	}
	return strings.Join(policies, "\n")
}

func getPolicy(ctx context.Context, client *api.Client, region, name string) string {
	response, err := client.GetRAMPolicy(ctx, region, name, "Custom")
	if err != nil {
		return err.Error()
	}
	return response.DefaultPolicyVersion.PolicyDocument
}
