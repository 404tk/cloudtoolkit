package iam

import (
	"strings"

	"github.com/aws/aws-sdk-go/service/iam"
)

func listAttachedUserPolicies(client *iam.IAM, name string) string {
	input := &iam.ListAttachedUserPoliciesInput{UserName: &name}
	resp, err := client.ListAttachedUserPolicies(input)
	if err != nil {
		return ""
	}
	policies := []string{}
	for _, p := range resp.AttachedPolicies {
		policies = append(policies, *p.PolicyName)
	}
	return strings.Join(policies, "\n")
}
