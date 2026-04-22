package replay

import (
	"fmt"
	"strings"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type hostFixture struct {
	InstanceID  string
	Hostname    string
	Status      string
	OSType      string
	PublicIP    string
	PrivateIP   string
	Region      string
	AgentStatus string
}

type iamUserFixture struct {
	UserName      string
	AccountID     int64
	CreateDate    string
	LoginAllowed  bool
	LastLoginDate string
}

type bucketFixture struct {
	Name    string
	Region  string
	Objects []bucketObjectFixture
}

type bucketObjectFixture struct {
	Key  string
	Size int64
}

type dnsZoneFixture struct {
	ID      int64
	Name    string
	Records []dnsRecordFixture
}

type dnsRecordFixture struct {
	Host   string
	Type   string
	Value  string
	Enable bool
}

type mysqlFixture struct {
	InstanceID string
	Region     string
	Version    string
	PublicHost string
	PrivateIP  string
	Port       string
}

type postgresFixture struct {
	InstanceID string
	Region     string
	Version    string
	PrivateIP  string
	Port       string
}

type sqlServerFixture struct {
	InstanceID string
	Region     string
	Version    string
	PrimaryIP  string
	Port       string
}

type shellTargetFixture struct {
	Commands map[string][]string
}

const (
	demoProject   = "demo"
	demoAccountID = int64(2101253872)
)

var demoCredentials = loadDemoCredentials()

var demoHosts = []hostFixture{
	{
		InstanceID:  "i-volc001",
		Hostname:    "app-01",
		Status:      "Running",
		OSType:      "Linux",
		PublicIP:    "203.0.113.41",
		PrivateIP:   "172.16.10.41",
		Region:      "cn-beijing",
		AgentStatus: "Running",
	},
	{
		InstanceID:  "i-volc002",
		Hostname:    "jump-01",
		Status:      "Running",
		OSType:      "Linux",
		PublicIP:    "203.0.113.42",
		PrivateIP:   "172.16.10.42",
		Region:      "cn-guangzhou",
		AgentStatus: "Running",
	},
}

var demoIAMUsers = []iamUserFixture{
	{
		UserName:      "admin",
		AccountID:     10001,
		CreateDate:    "20260420T090000Z",
		LoginAllowed:  true,
		LastLoginDate: "20260422T100000Z",
	},
	{
		UserName:      "audit",
		AccountID:     10002,
		CreateDate:    "20260420T093000Z",
		LoginAllowed:  false,
		LastLoginDate: "",
	},
}

var demoBuckets = []bucketFixture{
	{
		Name:   "volc-tos",
		Region: "cn-beijing",
		Objects: []bucketObjectFixture{
			{Key: "audit/2026-04-22/events.json", Size: 14541},
			{Key: "configs/app-prod.yaml", Size: 2232},
			{Key: "exports/inventory-2026-04-22.csv", Size: 1069548},
		},
	},
}

var demoZones = []dnsZoneFixture{
	{
		ID:   101,
		Name: "demo.local",
		Records: []dnsRecordFixture{
			{Host: "@", Type: "A", Value: "203.0.113.41", Enable: true},
			{Host: "www", Type: "CNAME", Value: "app-01.vol.local", Enable: true},
		},
	},
}

var demoMySQLInstances = []mysqlFixture{
	{
		InstanceID: "mysql-001",
		Region:     "cn-beijing",
		Version:    "MySQL_8_0",
		PublicHost: "mysql-001.rds.vol.local",
		PrivateIP:  "10.0.1.21",
		Port:       "3306",
	},
}

var demoPostgresInstances = []postgresFixture{
	{
		InstanceID: "pg-001",
		Region:     "cn-guangzhou",
		Version:    "PostgreSQL_14",
		PrivateIP:  "10.0.2.32",
		Port:       "5432",
	},
}

var demoSQLServerInstances = []sqlServerFixture{
	{
		InstanceID: "sqlserver-001",
		Region:     "cn-beijing",
		Version:    "2019",
		PrimaryIP:  "10.0.3.43",
		Port:       "1433",
	},
}

var demoShellTargets = map[string]shellTargetFixture{
	"i-volc001": {
		Commands: map[string][]string{
			"whoami": {"root"},
			"pwd":    {"/root"},
			"ls":     {"audit.log", "ctk", "tmp"},
			"id":     {"uid=0(root) gid=0(root) groups=0(root)"},
		},
	},
	"i-volc002": {
		Commands: map[string][]string{
			"whoami": {"ubuntu"},
			"pwd":    {"/home/ubuntu"},
			"ls":     {"app", "history.log", "tmp"},
			"id":     {"uid=1000(ubuntu) gid=1000(ubuntu) groups=1000(ubuntu)"},
		},
	},
}

func demoRegions() []string {
	seen := make(map[string]struct{})
	regions := make([]string, 0, len(demoHosts))
	for _, host := range demoHosts {
		if _, ok := seen[host.Region]; ok {
			continue
		}
		seen[host.Region] = struct{}{}
		regions = append(regions, host.Region)
	}
	return regions
}

func findHost(instanceID string) (hostFixture, bool) {
	for _, host := range demoHosts {
		if host.InstanceID == strings.TrimSpace(instanceID) {
			return host, true
		}
	}
	return hostFixture{}, false
}

func hostsForRegion(region string) []hostFixture {
	items := make([]hostFixture, 0, len(demoHosts))
	for _, host := range demoHosts {
		if host.Region == strings.TrimSpace(region) {
			items = append(items, host)
		}
	}
	return items
}

func findUser(userName string) (iamUserFixture, bool) {
	for _, user := range demoIAMUsers {
		if user.UserName == strings.TrimSpace(userName) {
			return user, true
		}
	}
	return iamUserFixture{}, false
}

func findBucket(name string) (bucketFixture, bool) {
	for _, bucket := range demoBuckets {
		if bucket.Name == strings.TrimSpace(name) {
			return bucket, true
		}
	}
	return bucketFixture{}, false
}

func listBucketsForRegion(region string) []bucketFixture {
	region = strings.TrimSpace(region)
	if region == "" {
		return append([]bucketFixture(nil), demoBuckets...)
	}
	items := make([]bucketFixture, 0, len(demoBuckets))
	for _, bucket := range demoBuckets {
		if bucket.Region == region {
			items = append(items, bucket)
		}
	}
	return items
}

func findZone(id int64) (dnsZoneFixture, bool) {
	for _, zone := range demoZones {
		if zone.ID == id {
			return zone, true
		}
	}
	return dnsZoneFixture{}, false
}

func mysqlForRegion(region string) []mysqlFixture {
	return filterMySQL(region)
}

func postgresForRegion(region string) []postgresFixture {
	return filterPostgres(region)
}

func sqlServerForRegion(region string) []sqlServerFixture {
	return filterSQLServer(region)
}

func filterMySQL(region string) []mysqlFixture {
	items := make([]mysqlFixture, 0, len(demoMySQLInstances))
	for _, item := range demoMySQLInstances {
		if item.Region == strings.TrimSpace(region) {
			items = append(items, item)
		}
	}
	return items
}

func filterPostgres(region string) []postgresFixture {
	items := make([]postgresFixture, 0, len(demoPostgresInstances))
	for _, item := range demoPostgresInstances {
		if item.Region == strings.TrimSpace(region) {
			items = append(items, item)
		}
	}
	return items
}

func filterSQLServer(region string) []sqlServerFixture {
	items := make([]sqlServerFixture, 0, len(demoSQLServerInstances))
	for _, item := range demoSQLServerInstances {
		if item.Region == strings.TrimSpace(region) {
			items = append(items, item)
		}
	}
	return items
}

func projectDisplayName() string {
	return fmt.Sprintf("%s(%d)", demoProject, demoAccountID)
}

func shellOutput(instanceID, command string) string {
	target, ok := demoShellTargets[strings.TrimSpace(instanceID)]
	if !ok {
		return ""
	}
	command = strings.TrimSpace(command)
	if lines, ok := target.Commands[command]; ok {
		return strings.Join(lines, "\n")
	}
	return fmt.Sprintf("command not found: %s", command)
}

func loadDemoCredentials() demoreplay.Credentials {
	creds, ok := demoreplay.CredentialsFor("volcengine")
	if !ok {
		return demoreplay.Credentials{}
	}
	return creds
}
