package api

import (
	"fmt"
	"strings"
)

const DefaultRegion = "cn-hangzhou"

type endpointResolution struct {
	Host        string
	TryLocation bool
}

func NormalizeRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" || strings.EqualFold(region, "all") {
		return DefaultRegion
	}
	return region
}

func resolveEndpointHost(product, region string) (endpointResolution, error) {
	switch strings.ToLower(strings.TrimSpace(product)) {
	case "sts":
		switch region {
		case "cn-north-2-gov-1":
			return endpointResolution{Host: "sts-vpc.cn-north-2-gov-1.aliyuncs.com"}, nil
		case "cn-shenzhen-finance-1":
			return endpointResolution{Host: "sts-vpc.cn-shenzhen-finance-1.aliyuncs.com"}, nil
		default:
			return endpointResolution{Host: "sts.aliyuncs.com"}, nil
		}
	case "bssopenapi":
		if _, ok := bssOverseasRegions[region]; ok {
			return endpointResolution{Host: "business.ap-southeast-1.aliyuncs.com"}, nil
		}
		return endpointResolution{Host: "business.aliyuncs.com"}, nil
	case "alidns":
		return endpointResolution{Host: "alidns.aliyuncs.com"}, nil
	case "ram":
		return endpointResolution{Host: "ram.aliyuncs.com"}, nil
	case "ecs":
		if host, ok := ecsRegionalEndpoints[region]; ok {
			return endpointResolution{Host: host}, nil
		}
		return endpointResolution{
			Host:        ecsGlobalEndpoint,
			TryLocation: true,
		}, nil
	case "rds":
		return endpointResolution{Host: "rds.aliyuncs.com"}, nil
	case "dysmsapi":
		return endpointResolution{Host: "dysmsapi.aliyuncs.com"}, nil
	case "sas":
		return endpointResolution{Host: "tds.aliyuncs.com"}, nil
	default:
		return endpointResolution{}, fmt.Errorf("alibaba client: unsupported product %q", product)
	}
}

const ecsGlobalEndpoint = "ecs-cn-hangzhou.aliyuncs.com"

var ecsRegionalEndpoints = map[string]string{
	"ap-northeast-1": "ecs.ap-northeast-1.aliyuncs.com",
	"ap-south-1":     "ecs.ap-south-1.aliyuncs.com",
	"ap-southeast-1": "ecs-cn-hangzhou.aliyuncs.com",
	"ap-southeast-2": "ecs.ap-southeast-2.aliyuncs.com",
	"ap-southeast-3": "ecs.ap-southeast-3.aliyuncs.com",
	"ap-southeast-5": "ecs.ap-southeast-5.aliyuncs.com",
	"cn-beijing":     "ecs-cn-hangzhou.aliyuncs.com",
	"cn-hangzhou":    "ecs-cn-hangzhou.aliyuncs.com",
	"cn-hongkong":    "ecs-cn-hangzhou.aliyuncs.com",
	"cn-huhehaote":   "ecs.cn-huhehaote.aliyuncs.com",
	"cn-qingdao":     "ecs-cn-hangzhou.aliyuncs.com",
	"cn-shanghai":    "ecs-cn-hangzhou.aliyuncs.com",
	"cn-shenzhen":    "ecs-cn-hangzhou.aliyuncs.com",
	"cn-zhangjiakou": "ecs.cn-zhangjiakou.aliyuncs.com",
	"eu-central-1":   "ecs.eu-central-1.aliyuncs.com",
	"eu-west-1":      "ecs.eu-west-1.aliyuncs.com",
	"me-east-1":      "ecs.me-east-1.aliyuncs.com",
	"us-east-1":      "ecs-cn-hangzhou.aliyuncs.com",
	"us-west-1":      "ecs-cn-hangzhou.aliyuncs.com",
}

var bssOverseasRegions = map[string]struct{}{
	"us-west-1":          {},
	"rus-west-1-pop":     {},
	"ap-northeast-2":     {},
	"ap-northeast-1":     {},
	"ap-southeast-1":     {},
	"ap-southeast-2":     {},
	"ap-southeast-3":     {},
	"ap-southeast-5":     {},
	"us-east-1":          {},
	"eu-central-1":       {},
	"eu-west-1":          {},
	"eu-west-1-oxs":      {},
	"me-east-1":          {},
	"ap-south-1":         {},
	"ap-northeast-2-pop": {},
}
