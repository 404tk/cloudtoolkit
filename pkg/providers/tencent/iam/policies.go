package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

var policy_infos map[string]string

func listAttachedUserAllPolicies(ctx context.Context, client *api.Client, uin uint64) string {
	resp, err := client.ListAttachedUserAllPolicies(ctx, uin, 1, 20, 0)
	if err != nil {
		return ""
	}
	policies := []string{}
	for _, p := range resp.Response.PolicyList {
		policyName := derefString(p.PolicyName)
		policies = append(policies, policyName)
		if derefString(p.StrategyType) == "1" {
			if _, ok := policy_infos[policyName]; !ok {
				details := getPolicy(ctx, client, derefString(p.PolicyID))
				policy_infos[policyName] = details
				msg := fmt.Sprintf("Found Custom Policy %s: %s", policyName, details)
				logger.Warning(msg)
			}
		}
	}
	return strings.Join(policies, "\n")
}

func getPolicy(ctx context.Context, client *api.Client, pid string) string {
	var policyID uint64
	_, err := fmt.Sscanf(pid, "%d", &policyID)
	if err != nil {
		return err.Error()
	}
	response, err := client.GetPolicy(ctx, policyID)
	if err != nil {
		return err.Error()
	}
	return derefString(response.Response.PolicyDocument)
}
