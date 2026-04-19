package api

import "testing"

func TestResolveEndpoint(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		region    string
		siteStack string
		want      string
	}{
		{name: "global iam", service: "iam", region: "cn-beijing", want: "https://iam.volcengineapi.com"},
		{name: "global billing", service: "billing", region: "", want: "https://billing.volcengineapi.com"},
		{name: "regional ecs", service: "ecs", region: "cn-shanghai", want: "https://ecs.cn-shanghai.volcengineapi.com"},
		{name: "fallback open", service: "ecs", region: "", want: "https://open.volcengineapi.com"},
		{name: "custom stack", service: "ecs", region: "cn-beijing", siteStack: "volcengine-api", want: "https://ecs.cn-beijing.volcengine-api.com"},
	}
	for _, tt := range tests {
		if got := ResolveEndpoint(tt.service, tt.region, tt.siteStack); got != tt.want {
			t.Fatalf("%s: ResolveEndpoint() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
