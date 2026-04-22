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
		{name: "regional rds mysql alias", service: "rds_mysql", region: "cn-beijing", want: "https://rds-mysql.cn-beijing.volcengineapi.com"},
		{name: "regional rds postgresql alias", service: "rds_postgresql", region: "cn-guangzhou", want: "https://rds-postgresql.cn-guangzhou.volcengineapi.com"},
		{name: "regional rds mssql alias", service: "rds_mssql", region: "cn-shanghai", want: "https://rds-mssql.cn-shanghai.volcengineapi.com"},
		{name: "fallback open", service: "ecs", region: "", want: "https://open.volcengineapi.com"},
		{name: "custom stack", service: "ecs", region: "cn-beijing", siteStack: "volcengine-api", want: "https://ecs.cn-beijing.volcengine-api.com"},
	}
	for _, tt := range tests {
		if got := ResolveEndpoint(tt.service, tt.region, tt.siteStack); got != tt.want {
			t.Fatalf("%s: ResolveEndpoint() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
