package replay

import "strings"

type uhostFixture struct {
	Name      string
	UHostID   string
	OsType    string
	State     string
	Region    string
	PrivateIP string
	PublicIP  string
}

var demoUHosts = []uhostFixture{
	{
		Name:      "ctk-demo-bastion",
		UHostID:   "uhost-001",
		OsType:    "Linux",
		State:     "Running",
		Region:    "cn-bj2",
		PrivateIP: "10.0.0.41",
		PublicIP:  "203.0.113.71",
	},
	{
		Name:      "ctk-demo-app",
		UHostID:   "uhost-002",
		OsType:    "Linux",
		State:     "Running",
		Region:    "cn-bj2",
		PrivateIP: "10.0.0.42",
	},
	{
		Name:      "ctk-demo-edge",
		UHostID:   "uhost-101",
		OsType:    "Linux",
		State:     "Running",
		Region:    "cn-sh2",
		PrivateIP: "10.10.0.51",
		PublicIP:  "203.0.113.72",
	},
}

func uhostsForRegion(region string) []uhostFixture {
	region = strings.TrimSpace(region)
	out := make([]uhostFixture, 0, len(demoUHosts))
	for _, host := range demoUHosts {
		if host.Region == region {
			out = append(out, host)
		}
	}
	return out
}

type bucketFixture struct {
	BucketName string
	Region     string
}

var demoBuckets = []bucketFixture{
	{BucketName: "ctk-validation-logs", Region: "cn-bj2"},
	{BucketName: "ctk-validation-archive", Region: "cn-sh2"},
}

func bucketsForRegion(region string) []bucketFixture {
	region = strings.TrimSpace(region)
	if region == "" || strings.EqualFold(region, "all") {
		return append([]bucketFixture(nil), demoBuckets...)
	}
	out := make([]bucketFixture, 0, len(demoBuckets))
	for _, bucket := range demoBuckets {
		if bucket.Region == region {
			out = append(out, bucket)
		}
	}
	return out
}

type udbFixture struct {
	DBID         string
	Name         string
	DBTypeID     string
	DBSubVersion string
	Port         int
	VirtualIP    string
	Region       string
	ClassType    string
}

var demoUDBInstances = []udbFixture{
	{
		DBID:         "udb-mysql-001",
		Name:         "ctk-demo-mysql",
		DBTypeID:     "mysql-8.0",
		DBSubVersion: "8.0.32",
		Port:         3306,
		VirtualIP:    "10.0.0.61",
		Region:       "cn-bj2",
		ClassType:    "sql",
	},
	{
		DBID:         "udb-pg-001",
		Name:         "ctk-demo-pg",
		DBTypeID:     "postgresql-14",
		DBSubVersion: "14.7",
		Port:         5432,
		VirtualIP:    "10.10.0.62",
		Region:       "cn-sh2",
		ClassType:    "postgresql",
	},
}

func udbForRegionAndClass(region, classType string) []udbFixture {
	region = strings.TrimSpace(region)
	classType = strings.TrimSpace(classType)
	out := make([]udbFixture, 0, len(demoUDBInstances))
	for _, item := range demoUDBInstances {
		if item.Region == region && item.ClassType == classType {
			out = append(out, item)
		}
	}
	return out
}

type dnsRecordFixture struct {
	Name      string
	Type      string
	Value     string
	IsEnabled int
}

type dnsZoneFixture struct {
	DNSZoneID   string
	DNSZoneName string
	Region      string
	Records     []dnsRecordFixture
}

var demoDNSZones = []dnsZoneFixture{
	{
		DNSZoneID:   "zone-ctkdemo",
		DNSZoneName: "demo.ctk.local",
		Region:      "cn-bj2",
		Records: []dnsRecordFixture{
			{Name: "@", Type: "A", Value: "203.0.113.71", IsEnabled: 1},
			{Name: "www", Type: "CNAME", Value: "demo.ctk.local", IsEnabled: 1},
		},
	},
}

func dnsZonesForRegion(region string) []dnsZoneFixture {
	region = strings.TrimSpace(region)
	out := make([]dnsZoneFixture, 0, len(demoDNSZones))
	for _, zone := range demoDNSZones {
		if zone.Region == region {
			out = append(out, zone)
		}
	}
	return out
}

func findDNSZone(id string) (dnsZoneFixture, bool) {
	id = strings.TrimSpace(id)
	for _, zone := range demoDNSZones {
		if zone.DNSZoneID == id {
			return zone, true
		}
	}
	return dnsZoneFixture{}, false
}

type subUserFixture struct {
	UserName    string
	DisplayName string
	Email       string
	Status      string
	CreatedAt   int64
}

var demoBaseSubUsers = []subUserFixture{
	{
		UserName:    "ctk-demo-readonly",
		DisplayName: "ctk-demo-readonly",
		Email:       "readonly@ctk.demo",
		Status:      "Active",
		CreatedAt:   1714435200,
	},
	{
		UserName:    "ctk-demo-bot",
		DisplayName: "ctk-demo-bot",
		Email:       "bot@ctk.demo",
		Status:      "Active",
		CreatedAt:   1714521600,
	},
}

var demoRegionList = []string{"cn-bj2", "cn-sh2", "hk", "us-ca"}
