package endpoint

import "testing"

func TestFor(t *testing.T) {
	cases := []struct {
		name    string
		service string
		region  string
		intl    bool
		want    string
	}{
		{"ecs cn", "ecs", "cn-north-4", false, "https://ecs.cn-north-4.myhuaweicloud.com"},
		{"ecs unknown region (no panic)", "ecs", "cn-zzz-99", false, "https://ecs.cn-zzz-99.myhuaweicloud.com"},
		{"rds ap", "rds", "ap-southeast-3", false, "https://rds.ap-southeast-3.myhuaweicloud.com"},
		{"iam default", "iam", "cn-north-1", false, "https://iam.cn-north-1.myhuaweicloud.com"},
		{"bss domestic", "bss", "any", false, "https://bss.myhuaweicloud.com"},
		{"bss intl", "bss", "any", true, "https://bss-intl.myhuaweicloud.com"},
		{"obs explicit", "obs", "cn-north-4", false, "https://obs.cn-north-4.myhuaweicloud.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := For(tc.service, tc.region, tc.intl)
			if got != tc.want {
				t.Errorf("For(%q, %q, %v) = %q, want %q", tc.service, tc.region, tc.intl, got, tc.want)
			}
		})
	}
}

// TestForDoesNotPanicOnUnknownRegion is the regression test for the original
// bug: region.ValueOf() from the official SDK panics on unknown regions. Our
// endpoint builder must never panic regardless of input.
func TestForDoesNotPanicOnUnknownRegion(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("endpoint.For panicked: %v", r)
		}
	}()
	inputs := []struct{ svc, region string }{
		{"ecs", "totally-made-up-region"},
		{"rds", ""},
		{"iam", "xx-yy-99"},
		{"unknown-service", "unknown-region"},
	}
	for _, in := range inputs {
		_ = For(in.svc, in.region, false)
		_ = For(in.svc, in.region, true)
	}
}
