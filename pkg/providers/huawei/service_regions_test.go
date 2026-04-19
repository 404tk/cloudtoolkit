package huawei

import (
	"reflect"
	"testing"

	huaweiauth "github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
)

func TestProviderServiceRegionsFiltersOnlyWhenRegionAll(t *testing.T) {
	p := &Provider{
		cred:    huaweiauth.New("ak", "sk", "all", false),
		regions: []string{"cn-north-4", "cn-east-201", "cn-east-3", "cn-north-219"},
	}

	if got, want := p.serviceRegions("ecs"), []string{"cn-north-4", "cn-east-3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("serviceRegions(ecs) = %v, want %v", got, want)
	}
	if got, want := p.serviceRegions("rds"), []string{"cn-north-4", "cn-east-3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("serviceRegions(rds) = %v, want %v", got, want)
	}
}

func TestProviderServiceRegionsKeepsExplicitRegionSelection(t *testing.T) {
	p := &Provider{
		cred:    huaweiauth.New("ak", "sk", "cn-east-201", false),
		regions: []string{"cn-east-201"},
	}

	if got, want := p.serviceRegions("rds"), []string{"cn-east-201"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("serviceRegions(rds) = %v, want %v", got, want)
	}
}
