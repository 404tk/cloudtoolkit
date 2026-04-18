package api

import (
	"fmt"
	"strings"
)

const DefaultRegion = "cn-hangzhou"

func NormalizeRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" || strings.EqualFold(region, "all") {
		return DefaultRegion
	}
	return region
}

func resolveEndpointHost(product, region string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(product)) {
	case "sts":
		switch region {
		case "cn-north-2-gov-1":
			return "sts-vpc.cn-north-2-gov-1.aliyuncs.com", nil
		case "cn-shenzhen-finance-1":
			return "sts-vpc.cn-shenzhen-finance-1.aliyuncs.com", nil
		default:
			return "sts.aliyuncs.com", nil
		}
	case "bssopenapi":
		if _, ok := bssOverseasRegions[region]; ok {
			return "business.ap-southeast-1.aliyuncs.com", nil
		}
		return "business.aliyuncs.com", nil
	case "alidns":
		return "alidns.aliyuncs.com", nil
	case "ram":
		return "ram.aliyuncs.com", nil
	case "ecs":
		return "ecs.aliyuncs.com", nil
	case "rds":
		return "rds.aliyuncs.com", nil
	case "dysmsapi":
		return "dysmsapi.aliyuncs.com", nil
	case "sas":
		return "tds.aliyuncs.com", nil
	default:
		return "", fmt.Errorf("alibaba client: unsupported product %q", product)
	}
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
