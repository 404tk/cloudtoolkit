package iam

import (
	"context"
	"strings"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

func listAttachedUserPolicies(ctx context.Context, client *iam.Client, name string) string {
	paginator := iam.NewListAttachedUserPoliciesPaginator(client, &iam.ListAttachedUserPoliciesInput{
		UserName: &name,
	})
	policies := []string{}
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return ""
		}
		for _, p := range resp.AttachedPolicies {
			policies = append(policies, awsv2.ToString(p.PolicyName))
		}
	}
	return strings.Join(policies, "\n")
}
