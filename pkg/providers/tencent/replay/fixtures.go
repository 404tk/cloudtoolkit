package replay

import (
	"fmt"
	"strings"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

type camPolicyFixture struct {
	ID           uint64
	Name         string
	StrategyType string
	Document     string
}

type camUserFixture struct {
	UIN          uint64
	Name         string
	ConsoleLogin bool
	CreateTime   string
	Policies     []camPolicyFixture
}

type cvmFixture struct {
	InstanceID   string
	InstanceName string
	State        string
	Region       string
	PublicIP     string
	PrivateIP    string
	OSName       string
}

type lighthouseFixture struct {
	InstanceID    string
	InstanceName  string
	State         string
	Region        string
	PublicAddress string
	PrivateIP     string
	PlatformType  string
}

type domainRecordFixture struct {
	Name   string
	Type   string
	Value  string
	Status string
}

type domainFixture struct {
	Name      string
	Status    string
	DNSStatus string
	Records   []domainRecordFixture
}

type mysqlFixture struct {
	InstanceID string
	Region     string
	Version    string
	WanStatus  int64
	WanDomain  string
	WanPort    int64
	VIP        string
	VPort      int64
}

type mariadbFixture struct {
	InstanceID string
	Region     string
	Version    string
	WanStatus  int64
	WanDomain  string
	WanPort    int64
	VIP        string
	VPort      int64
}

type postgresFixture struct {
	InstanceID    string
	Engine        string
	Version       string
	Region        string
	PublicAddress string
	PrivateIP     string
	Port          uint64
}

type sqlServerFixture struct {
	InstanceID   string
	VersionName  string
	Version      string
	Region       string
	DNSPodDomain string
	TgwWanVPort  int64
	VIP          string
	VPort        int64
}

type bucketObjectFixture struct {
	Key  string
	Size int64
}

type bucketFixture struct {
	Name         string
	Region       string
	CreationDate string
	Objects      []bucketObjectFixture
}

type shellTargetFixture struct {
	Commands map[string][]string
}

const (
	demoOwnerUIN = "100000001"
	demoCallerID = "qcs::cam::uin/100000001:uin/100000001"
)

var demoCredentials = loadDemoCredentials()

var demoRegions = []string{
	"ap-guangzhou",
	"ap-shanghai",
}

var demoPolicies = []camPolicyFixture{
	{
		ID:           1,
		Name:         "AdministratorAccess",
		StrategyType: "2",
		Document:     `{"version":"2.0","statement":[{"effect":"allow","action":"*","resource":"*"}]}`,
	},
	{
		ID:           200001,
		Name:         "AuditReadOnly",
		StrategyType: "1",
		Document:     `{"version":"2.0","statement":[{"effect":"allow","action":["cam:Get*","cam:List*","cvm:Describe*","dnspod:Describe*","cdb:*Describe*"],"resource":"*"}]}`,
	},
}

var demoCAMUsers = []camUserFixture{
	{
		UIN:          100000101,
		Name:         "admin",
		ConsoleLogin: true,
		CreateTime:   "2026-04-20 09:00:00",
		Policies: []camPolicyFixture{
			demoPolicies[0],
		},
	},
	{
		UIN:          100000102,
		Name:         "audit",
		ConsoleLogin: false,
		CreateTime:   "2026-04-20 09:15:00",
		Policies: []camPolicyFixture{
			demoPolicies[1],
		},
	},
}

var demoCVMInstances = []cvmFixture{
	{
		InstanceID:   "ins-cvm001",
		InstanceName: "cvm-01",
		State:        "RUNNING",
		Region:       "ap-guangzhou",
		PublicIP:     "203.0.113.31",
		PrivateIP:    "10.10.1.31",
		OSName:       "TencentOS Server 3.1",
	},
	{
		InstanceID:   "ins-cvm002",
		InstanceName: "cvm-02",
		State:        "RUNNING",
		Region:       "ap-shanghai",
		PublicIP:     "203.0.113.32",
		PrivateIP:    "10.10.2.32",
		OSName:       "Windows Server 2019 Datacenter",
	},
}

var demoLighthouseInstances = []lighthouseFixture{
	{
		InstanceID:    "lhins-001",
		InstanceName:  "light-01",
		State:         "RUNNING",
		Region:        "ap-guangzhou",
		PublicAddress: "203.0.113.41",
		PrivateIP:     "10.20.1.41",
		PlatformType:  "LINUX_UNIX",
	},
}

var demoDomains = []domainFixture{
	{
		Name:      "demo.local",
		Status:    "ENABLE",
		DNSStatus: "DNSSEC_DISABLE",
		Records: []domainRecordFixture{
			{Name: "@", Type: "A", Value: "203.0.113.31", Status: "ENABLE"},
			{Name: "www", Type: "CNAME", Value: "cvm-01.tx.local", Status: "ENABLE"},
		},
	},
}

var demoMySQLInstances = []mysqlFixture{
	{
		InstanceID: "mysql-001",
		Region:     "ap-guangzhou",
		Version:    "8.0",
		WanStatus:  1,
		WanDomain:  "mysql-001.ap-guangzhou.db.tx.local",
		WanPort:    3306,
	},
}

var demoMariaDBInstances = []mariadbFixture{
	{
		InstanceID: "mariadb-001",
		Region:     "ap-guangzhou",
		Version:    "10.6",
		WanStatus:  0,
		VIP:        "10.30.1.15",
		VPort:      3306,
	},
}

var demoPostgresInstances = []postgresFixture{
	{
		InstanceID:    "pg-01",
		Engine:        "PostgreSQL",
		Version:       "14",
		Region:        "ap-guangzhou",
		PublicAddress: "pg-01.ap-guangzhou.db.tx.local",
		PrivateIP:     "10.30.2.25",
		Port:          5432,
	},
}

var demoSQLServerInstances = []sqlServerFixture{
	{
		InstanceID:   "ss-01",
		VersionName:  "SQL Server",
		Version:      "2019",
		Region:       "ap-shanghai",
		DNSPodDomain: "ss-01.ap-shanghai.db.tx.local",
		TgwWanVPort:  1433,
	},
}

var demoBuckets = []bucketFixture{
	{
		Name:         "ctk-1300000001",
		Region:       "ap-guangzhou",
		CreationDate: "2026-04-20T08:00:00.000Z",
		Objects: []bucketObjectFixture{
			{Key: "audit/2026-04-22/events.json", Size: 14541},
			{Key: "configs/app-prod.yaml", Size: 2232},
			{Key: "exports/inventory-2026-04-22.csv", Size: 1069548},
		},
	},
}

var demoShellTargets = map[string]shellTargetFixture{
	"ins-cvm001": {
		Commands: map[string][]string{
			"whoami": {"ubuntu"},
			"pwd":    {"/home/ubuntu"},
			"ls":     {"app", "audit.log", "tmp"},
			"id":     {"uid=1000(ubuntu) gid=1000(ubuntu) groups=1000(ubuntu),27(sudo)"},
		},
	},
	"ins-cvm002": {
		Commands: map[string][]string{
			"whoami": {`corp\administrator`},
		},
	},
}

func loadDemoCredentials() demoreplay.Credentials {
	creds, ok := demoreplay.CredentialsFor("tencent")
	if !ok {
		return demoreplay.Credentials{}
	}
	return creds
}

func demoBalanceCents() int64 {
	return 102488
}

func listCAMUsers(created map[string]camUserFixture) []camUserFixture {
	users := make([]camUserFixture, 0, len(demoCAMUsers)+len(created))
	users = append(users, demoCAMUsers...)
	for _, user := range created {
		users = append(users, user)
	}
	return users
}

func findPolicy(policyID uint64) (camPolicyFixture, bool) {
	for _, policy := range demoPolicies {
		if policy.ID == policyID {
			return policy, true
		}
	}
	return camPolicyFixture{}, false
}

func findBucket(name string) (bucketFixture, bool) {
	for _, bucket := range demoBuckets {
		if bucket.Name == strings.TrimSpace(name) {
			return bucket, true
		}
	}
	return bucketFixture{}, false
}

func findDomain(name string) (domainFixture, bool) {
	for _, domain := range demoDomains {
		if domain.Name == strings.TrimSpace(name) {
			return domain, true
		}
	}
	return domainFixture{}, false
}

func cvmForRegion(region string) []cvmFixture {
	items := make([]cvmFixture, 0, len(demoCVMInstances))
	for _, instance := range demoCVMInstances {
		if instance.Region == strings.TrimSpace(region) {
			items = append(items, instance)
		}
	}
	return items
}

func lighthouseForRegion(region string) []lighthouseFixture {
	items := make([]lighthouseFixture, 0, len(demoLighthouseInstances))
	for _, instance := range demoLighthouseInstances {
		if instance.Region == strings.TrimSpace(region) {
			items = append(items, instance)
		}
	}
	return items
}

func mysqlForRegion(region string) []mysqlFixture {
	items := make([]mysqlFixture, 0, len(demoMySQLInstances))
	for _, instance := range demoMySQLInstances {
		if instance.Region == strings.TrimSpace(region) {
			items = append(items, instance)
		}
	}
	return items
}

func mariadbForRegion(region string) []mariadbFixture {
	items := make([]mariadbFixture, 0, len(demoMariaDBInstances))
	for _, instance := range demoMariaDBInstances {
		if instance.Region == strings.TrimSpace(region) {
			items = append(items, instance)
		}
	}
	return items
}

func postgresForRegion(region string) []postgresFixture {
	items := make([]postgresFixture, 0, len(demoPostgresInstances))
	for _, instance := range demoPostgresInstances {
		if instance.Region == strings.TrimSpace(region) {
			items = append(items, instance)
		}
	}
	return items
}

func sqlServerForRegion(region string) []sqlServerFixture {
	items := make([]sqlServerFixture, 0, len(demoSQLServerInstances))
	for _, instance := range demoSQLServerInstances {
		if instance.Region == strings.TrimSpace(region) {
			items = append(items, instance)
		}
	}
	return items
}

func bucketPage(objects []bucketObjectFixture, marker string, maxKeys int) ([]bucketObjectFixture, string, bool) {
	if maxKeys <= 0 {
		maxKeys = 1000
	}
	start := 0
	marker = strings.TrimSpace(marker)
	if marker != "" {
		for i, object := range objects {
			if object.Key == marker {
				start = i + 1
				break
			}
		}
	}
	if start >= len(objects) {
		return nil, "", false
	}
	end := start + maxKeys
	if end > len(objects) {
		end = len(objects)
	}
	isTruncated := end < len(objects)
	nextMarker := ""
	if isTruncated && end > start {
		nextMarker = objects[end-1].Key
	}
	return objects[start:end], nextMarker, isTruncated
}

func shellOutput(instanceID, command string) string {
	command = strings.TrimSpace(command)
	target, ok := demoShellTargets[strings.TrimSpace(instanceID)]
	if !ok {
		return fmt.Sprintf("command not found: %s", command)
	}
	if lines, ok := target.Commands[command]; ok {
		return strings.Join(lines, "\n")
	}
	return fmt.Sprintf("command not found: %s", command)
}
