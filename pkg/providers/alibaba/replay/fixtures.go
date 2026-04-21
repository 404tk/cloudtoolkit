package replay

import (
	"fmt"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/oss"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type ramPolicyFixture struct {
	Name     string
	Type     string
	Document string
}

type ramUserFixture struct {
	UserName       string
	UserID         string
	CreateDate     string
	LastLoginDate  string
	HasLogin       bool
	AttachedPolicy []ramPolicyFixture
}

type rdsInstanceFixture struct {
	InstanceID    string
	Engine        string
	EngineVersion string
	Region        string
	Address       string
	NetworkType   string
	DBNames       []string
}

type bucketFixture struct {
	Name    string
	Region  string
	Objects []oss.OSSObject
}

type logProjectFixture struct {
	ProjectName string
	Region      string
	Description string
	ModifiedAt  time.Time
}

type sasEventFixture struct {
	ID        string
	Name      string
	Affected  string
	API       string
	Status    int
	SourceIP  string
	AccessKey string
	Time      string
}

type shellTargetFixture struct {
	Commands map[string][]string
}

var demoRegions = []string{
	"cn-hangzhou",
	"cn-beijing",
}

var demoHosts = []schema.Host{
	{
		HostName:    "app-01",
		ID:          "i-demo001",
		State:       "Running",
		PublicIPv4:  "203.0.113.21",
		PrivateIpv4: "172.16.10.21",
		OSType:      "linux",
		DNSName:     "i-demo001.cn-hangzhou.demo.internal",
		Public:      true,
		Region:      "cn-hangzhou",
	},
	{
		HostName:    "jump-01",
		ID:          "i-demo002",
		State:       "Running",
		PublicIPv4:  "203.0.113.22",
		PrivateIpv4: "172.16.10.22",
		OSType:      "linux",
		DNSName:     "i-demo002.cn-beijing.demo.internal",
		Public:      true,
		Region:      "cn-beijing",
	},
}

var demoDomains = []schema.Domain{
	{
		DomainName: "demo.local",
		Records: []schema.Record{
			{
				RR:     "www",
				Type:   "CNAME",
				Value:  "ctk-demo.oss-cn-hangzhou.aliyuncs.com",
				Status: "ENABLE",
			},
			{
				RR:     "api",
				Type:   "A",
				Value:  "203.0.113.21",
				Status: "ENABLE",
			},
		},
	},
}

var demoRAMUsers = []ramUserFixture{
	{
		UserName:      "demo",
		UserID:        "235000000000000001",
		CreateDate:    "2026-03-10T11:00:00+08:00",
		LastLoginDate: "2026-04-20T09:12:00+08:00",
		HasLogin:      true,
		AttachedPolicy: []ramPolicyFixture{
			{Name: "AdministratorAccess", Type: "System"},
		},
	},
	{
		UserName:   "audit",
		UserID:     "235000000000000002",
		CreateDate: "2026-03-11T16:40:00+08:00",
		HasLogin:   false,
		AttachedPolicy: []ramPolicyFixture{
			{Name: "AliyunActionTrailFullAccess", Type: "System"},
		},
	},
}

var demoRDSInstances = []rdsInstanceFixture{
	{
		InstanceID:    "rm-demo001",
		Engine:        "MySQL",
		EngineVersion: "8.0",
		Region:        "cn-hangzhou",
		Address:       "rm-demo001.mysql.rds.aliyuncs.com",
		NetworkType:   "VPC",
		DBNames:       []string{"appdb"},
	},
	{
		InstanceID:    "pg-demo002",
		Engine:        "PostgreSQL",
		EngineVersion: "14",
		Region:        "cn-beijing",
		Address:       "pg-demo002.pg.rds.aliyuncs.com",
		NetworkType:   "VPC",
		DBNames:       []string{"analytics"},
	},
}

var demoBuckets = []bucketFixture{
	{
		Name:   "ctk-demo",
		Region: "cn-hangzhou",
		Objects: []oss.OSSObject{
			{Key: "audit/2026-04-20/events.json", Size: 14541},
			{Key: "configs/app-prod.yaml", Size: 2232},
			{Key: "exports/inventory-2026-04-20.csv", Size: 1069548},
		},
	},
}

var demoLogProjects = []logProjectFixture{
	{
		ProjectName: "actiontrail-demo",
		Region:      "cn-hangzhou",
		Description: "all-region trail",
		ModifiedAt:  time.Date(2026, time.April, 20, 9, 20, 0, 0, time.FixedZone("CST", 8*3600)),
	},
	{
		ProjectName: "sls-demo-audit",
		Region:      "cn-beijing",
		Description: "indexed security events",
		ModifiedAt:  time.Date(2026, time.April, 20, 9, 21, 0, 0, time.FixedZone("CST", 8*3600)),
	},
}

var demoSASEvents = []sasEventFixture{
	{
		ID:        "ev-0001",
		Name:      "Create RAM User",
		Affected:  "demo-security-admin",
		API:       "ram:CreateUser",
		Status:    32,
		SourceIP:  "203.0.113.10",
		AccessKey: DemoAccessKeyID,
		Time:      "2026-04-20T09:10:11+08:00",
	},
	{
		ID:        "ev-0002",
		Name:      "Get Bucket Info",
		Affected:  "ctk-demo-bucket",
		API:       "oss:GetBucketInfo",
		Status:    16,
		SourceIP:  "203.0.113.10",
		AccessKey: DemoAccessKeyID,
		Time:      "2026-04-20T09:12:53+08:00",
	},
	{
		ID:        "ev-0003",
		Name:      "Run ECS Command",
		Affected:  "i-demoali001",
		API:       "ecs:RunCommand",
		Status:    32,
		SourceIP:  "203.0.113.10",
		AccessKey: DemoAccessKeyID,
		Time:      "2026-04-20T09:15:02+08:00",
	},
}

var demoShellTargets = map[string]shellTargetFixture{
	"i-demo001": {
		Commands: map[string][]string{
			"whoami": {"ubuntu"},
			"pwd":    {"/home/ubuntu"},
			"ls":     {"app", "audit.log", "tmp"},
			"id":     {"uid=1000(ubuntu) gid=1000(ubuntu) groups=1000(ubuntu),10(wheel)"},
		},
	},
	"i-demo002": {
		Commands: map[string][]string{
			"whoami": {"root"},
			"pwd":    {"/root"},
			"ls":     {"authorized_keys", "history.log"},
			"id":     {"uid=0(root) gid=0(root) groups=0(root)"},
		},
	},
}

func demoCallerArn() string {
	return "acs:ram::235000000000000001:user/demo"
}

func demoAccountAlias() string {
	return "ctk-demo"
}

func demoBalanceAmount() string {
	return "1024.88"
}

func demoSMSDailySize() int64 {
	return 42
}

func demoSMSSigns() []map[string]string {
	return []map[string]string{
		{
			"SignName":     "CTK-DEMO",
			"AuditStatus":  "AUDIT_STATE_PASS",
			"BusinessType": "verification",
		},
	}
}

func demoSMSTemplates() []map[string]string {
	return []map[string]string{
		{
			"TemplateName":    "login-alert",
			"AuditStatus":     "AUDIT_STATE_PASS",
			"TemplateContent": "demo validation login alert",
		},
	}
}

func hostsForRegion(region string) []schema.Host {
	return filterHostsByRegion(demoHosts, region)
}

func databasesForRegion(region string) []rdsInstanceFixture {
	if strings.TrimSpace(region) == "" {
		return append([]rdsInstanceFixture(nil), demoRDSInstances...)
	}
	items := make([]rdsInstanceFixture, 0, len(demoRDSInstances))
	for _, item := range demoRDSInstances {
		if item.Region == region {
			items = append(items, item)
		}
	}
	return items
}

func logProjectsForRegion(region string) []logProjectFixture {
	if strings.TrimSpace(region) == "" {
		return append([]logProjectFixture(nil), demoLogProjects...)
	}
	items := make([]logProjectFixture, 0, len(demoLogProjects))
	for _, item := range demoLogProjects {
		if item.Region == region {
			items = append(items, item)
		}
	}
	return items
}

func filterHostsByRegion(items []schema.Host, region string) []schema.Host {
	if strings.TrimSpace(region) == "" {
		return append([]schema.Host(nil), items...)
	}
	filtered := make([]schema.Host, 0, len(items))
	for _, item := range items {
		if item.Region == region {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func findRAMUser(userName string) (ramUserFixture, bool) {
	for _, user := range demoRAMUsers {
		if user.UserName == strings.TrimSpace(userName) {
			return user, true
		}
	}
	return ramUserFixture{}, false
}

func findBucket(name string) (bucketFixture, bool) {
	for _, bucket := range demoBuckets {
		if bucket.Name == strings.TrimSpace(name) {
			return bucket, true
		}
	}
	return bucketFixture{}, false
}

func findDomain(name string) (schema.Domain, bool) {
	for _, domain := range demoDomains {
		if domain.DomainName == strings.TrimSpace(name) {
			return domain, true
		}
	}
	return schema.Domain{}, false
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
