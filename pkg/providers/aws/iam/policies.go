package iam

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
)

func listAttachedUserPolicies(ctx context.Context, client *api.Client, region, name string) string {
	policies, err := paginate.Fetch[api.AttachedUserPolicy, string](ctx, func(ctx context.Context, marker string) (paginate.Page[api.AttachedUserPolicy, string], error) {
		resp, err := client.ListAttachedUserPolicies(ctx, region, name, marker)
		if err != nil {
			return paginate.Page[api.AttachedUserPolicy, string]{}, err
		}
		return paginate.Page[api.AttachedUserPolicy, string]{
			Items: resp.Policies,
			Next:  resp.Marker,
			Done:  !resp.IsTruncated || strings.TrimSpace(resp.Marker) == "",
		}, nil
	})
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(policies))
	for _, policy := range policies {
		names = append(names, policy.PolicyName)
	}
	return strings.Join(names, "\n")
}
