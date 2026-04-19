package huawei

var (
	ecsSupportedRegions = regionSet(
		"cn-north-1",
		"cn-north-4",
		"cn-south-1",
		"cn-east-2",
		"cn-east-3",
		"cn-southwest-2",
		"ap-southeast-1",
		"ap-southeast-2",
		"ap-southeast-3",
		"af-south-1",
		"sa-brazil-1",
		"la-north-2",
		"cn-south-4",
		"na-mexico-1",
		"la-south-2",
		"cn-south-2",
		"cn-north-9",
		"cn-north-2",
		"ap-southeast-4",
		"tr-west-1",
		"me-east-1",
		"ae-ad-1",
		"cn-east-4",
		"eu-west-101",
		"cn-east-5",
		"eu-west-0",
		"my-kualalumpur-1",
		"af-north-1",
		"ru-moscow-1",
		"ap-southeast-5",
		"cn-north-11",
		"cn-north-12",
		"cn-southwest-3",
	)
	rdsSupportedRegions = regionSet(
		"af-south-1",
		"cn-north-4",
		"cn-north-1",
		"cn-east-2",
		"cn-east-3",
		"cn-east-5",
		"cn-east-4",
		"cn-south-1",
		"cn-southwest-2",
		"ap-southeast-2",
		"ap-southeast-1",
		"ap-southeast-3",
		"ru-northwest-2",
		"sa-brazil-1",
		"la-north-2",
		"cn-south-2",
		"na-mexico-1",
		"la-south-2",
		"cn-north-9",
		"cn-north-2",
		"tr-west-1",
		"ap-southeast-4",
		"ap-southeast-5",
		"ae-ad-1",
		"eu-west-101",
		"eu-west-0",
		"my-kualalumpur-1",
		"ru-moscow-1",
		"me-east-1",
		"af-north-1",
		"cn-north-12",
		"cn-south-4",
		"cn-southwest-3",
		"cn-north-11",
	)
)

func regionSet(regions ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(regions))
	for _, region := range regions {
		set[region] = struct{}{}
	}
	return set
}

func (p *Provider) serviceRegions(service string) []string {
	if p == nil || p.cred.Region != "all" {
		return p.regions
	}

	var supported map[string]struct{}
	switch service {
	case "ecs":
		supported = ecsSupportedRegions
	case "rds":
		supported = rdsSupportedRegions
	default:
		return p.regions
	}

	filtered := make([]string, 0, len(p.regions))
	for _, region := range p.regions {
		if _, ok := supported[region]; ok {
			filtered = append(filtered, region)
		}
	}
	return filtered
}
