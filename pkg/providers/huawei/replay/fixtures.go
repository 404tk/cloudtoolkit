package replay

import "strings"

type regionFixture struct {
	ID string
}

var demoRegions = []regionFixture{
	{ID: "cn-north-4"},
	{ID: "cn-east-3"},
	{ID: "cn-south-1"},
	{ID: "ap-southeast-1"},
}

type projectFixture struct {
	ID       string
	Name     string
	DomainID string
}

var demoProjects = []projectFixture{
	{ID: "06f1d2dca680f0a02fa4c01acc0e1001", Name: "cn-north-4", DomainID: demoDomainID},
	{ID: "06f1d2dca680f0a02fa4c01acc0e1002", Name: "cn-east-3", DomainID: demoDomainID},
	{ID: "06f1d2dca680f0a02fa4c01acc0e1003", Name: "cn-south-1", DomainID: demoDomainID},
	{ID: "06f1d2dca680f0a02fa4c01acc0e1004", Name: "ap-southeast-1", DomainID: demoDomainID},
}

func findProject(name string) (projectFixture, bool) {
	name = strings.TrimSpace(name)
	for _, project := range demoProjects {
		if project.Name == name {
			return project, true
		}
	}
	return projectFixture{}, false
}

func findProjectByID(id string) (projectFixture, bool) {
	id = strings.TrimSpace(id)
	for _, project := range demoProjects {
		if project.ID == id {
			return project, true
		}
	}
	return projectFixture{}, false
}

type iamUserFixture struct {
	ID       string
	Name     string
	Enabled  bool
	DomainID string
}

var demoBaseIAMUsers = []iamUserFixture{
	{
		ID:       demoUserID,
		Name:     demoUserName,
		Enabled:  true,
		DomainID: demoDomainID,
	},
	{
		ID:       "06f1d2dca680f0a02fa4c01acc0e0100",
		Name:     "ctk-demo-readonly",
		Enabled:  true,
		DomainID: demoDomainID,
	},
	{
		ID:       "06f1d2dca680f0a02fa4c01acc0e0101",
		Name:     "ctk-demo-bot",
		Enabled:  false,
		DomainID: demoDomainID,
	},
}

type iamGroupFixture struct {
	ID   string
	Name string
}

var demoIAMGroups = []iamGroupFixture{
	{ID: "06f1d2dca680f0a02fa4c01acc0e0g01", Name: "admin"},
	{ID: "06f1d2dca680f0a02fa4c01acc0e0g02", Name: "readonly"},
}

type ecsHostFixture struct {
	ID        string
	Name      string
	Status    string
	Region    string
	PublicIP  string
	PrivateIP string
}

var demoECSHosts = []ecsHostFixture{
	{
		ID:        "0f001",
		Name:      "ctk-demo-bastion",
		Status:    "ACTIVE",
		Region:    "cn-north-4",
		PublicIP:  "203.0.113.61",
		PrivateIP: "192.168.10.61",
	},
	{
		ID:        "0f002",
		Name:      "ctk-demo-app",
		Status:    "ACTIVE",
		Region:    "cn-north-4",
		PrivateIP: "192.168.10.62",
	},
	{
		ID:        "0f101",
		Name:      "ctk-demo-edge",
		Status:    "ACTIVE",
		Region:    "cn-east-3",
		PublicIP:  "203.0.113.71",
		PrivateIP: "192.168.20.71",
	},
	{
		ID:        "0f201",
		Name:      "ctk-demo-batch",
		Status:    "SHUTOFF",
		Region:    "cn-south-1",
		PrivateIP: "192.168.30.81",
	},
}

func ecsHostsForRegion(region string) []ecsHostFixture {
	region = strings.TrimSpace(region)
	items := make([]ecsHostFixture, 0, len(demoECSHosts))
	for _, host := range demoECSHosts {
		if host.Region == region {
			items = append(items, host)
		}
	}
	return items
}

type rdsInstanceFixture struct {
	ID        string
	Engine    string
	Version   string
	Region    string
	Port      int32
	PrivateIP string
	PublicIP  string
}

var demoRDSInstances = []rdsInstanceFixture{
	{
		ID:        "rds-mysql-001",
		Engine:    "MySQL",
		Version:   "8.0",
		Region:    "cn-north-4",
		Port:      3306,
		PrivateIP: "192.168.50.21",
	},
	{
		ID:        "rds-postgres-001",
		Engine:    "PostgreSQL",
		Version:   "14",
		Region:    "cn-east-3",
		Port:      5432,
		PrivateIP: "192.168.60.21",
	},
}

func rdsInstancesForRegion(region string) []rdsInstanceFixture {
	region = strings.TrimSpace(region)
	items := make([]rdsInstanceFixture, 0, len(demoRDSInstances))
	for _, item := range demoRDSInstances {
		if item.Region == region {
			items = append(items, item)
		}
	}
	return items
}

type obsObjectFixture struct {
	Key          string
	Size         int64
	LastModified string
	StorageClass string
}

type obsBucketFixture struct {
	Name    string
	Region  string
	Objects []obsObjectFixture
}

var demoOBSBuckets = []obsBucketFixture{
	{
		Name:   "ctk-validation-logs",
		Region: "cn-north-4",
		Objects: []obsObjectFixture{
			{Key: "audit/2026-04-20.log", Size: 14820, LastModified: "2026-04-20T23:59:00.000Z", StorageClass: "STANDARD"},
			{Key: "audit/2026-04-21.log", Size: 15010, LastModified: "2026-04-21T23:59:00.000Z", StorageClass: "STANDARD"},
			{Key: "audit/2026-04-22.log", Size: 12940, LastModified: "2026-04-22T23:59:00.000Z", StorageClass: "STANDARD"},
		},
	},
	{
		Name:   "ctk-validation-archive",
		Region: "cn-east-3",
		Objects: []obsObjectFixture{
			{Key: "archive/2026Q1.tar.gz", Size: 1048576, LastModified: "2026-04-01T03:00:00.000Z", StorageClass: "COLD"},
		},
	},
}

func findOBSBucket(name string) (obsBucketFixture, bool) {
	name = strings.TrimSpace(name)
	for _, bucket := range demoOBSBuckets {
		if bucket.Name == name {
			return bucket, true
		}
	}
	return obsBucketFixture{}, false
}
