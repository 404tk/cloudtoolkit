package replay

import "strings"

type instanceFixture struct {
	Name      string
	Hostname  string
	Zone      string
	Status    string
	PrivateIP string
	PublicIP  string
}

var demoZones = []string{"us-central1-a", "us-east1-b", "asia-east1-a"}

var demoInstances = []instanceFixture{
	{
		Name:      "ctk-demo-bastion",
		Hostname:  "ctk-demo-bastion.c.ctk-demo-project.internal",
		Zone:      "us-central1-a",
		Status:    "RUNNING",
		PrivateIP: "10.10.0.21",
		PublicIP:  "203.0.113.41",
	},
	{
		Name:      "ctk-demo-app",
		Hostname:  "ctk-demo-app.c.ctk-demo-project.internal",
		Zone:      "us-central1-a",
		Status:    "RUNNING",
		PrivateIP: "10.10.0.22",
	},
	{
		Name:      "ctk-demo-edge",
		Hostname:  "ctk-demo-edge.c.ctk-demo-project.internal",
		Zone:      "us-east1-b",
		Status:    "RUNNING",
		PrivateIP: "10.20.0.31",
		PublicIP:  "203.0.113.51",
	},
}

func instancesForZone(zone string) []instanceFixture {
	zone = strings.TrimSpace(zone)
	out := make([]instanceFixture, 0, len(demoInstances))
	for _, inst := range demoInstances {
		if inst.Zone == zone {
			out = append(out, inst)
		}
	}
	return out
}

type serviceAccountFixture struct {
	Name           string
	UniqueID       string
	Email          string
	DisplayName    string
	OAuth2ClientID string
}

var demoServiceAccounts = []serviceAccountFixture{
	{
		Name:           "projects/ctk-demo-project/serviceAccounts/ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
		UniqueID:       "100000000000000000001",
		Email:          "ctk-demo@ctk-demo-project.iam.gserviceaccount.com",
		DisplayName:    "ctk-demo-admin",
		OAuth2ClientID: "100000000000000000001",
	},
	{
		Name:           "projects/ctk-demo-project/serviceAccounts/ctk-readonly@ctk-demo-project.iam.gserviceaccount.com",
		UniqueID:       "100000000000000000002",
		Email:          "ctk-readonly@ctk-demo-project.iam.gserviceaccount.com",
		DisplayName:    "ctk-readonly",
		OAuth2ClientID: "100000000000000000002",
	},
}

type managedZoneFixture struct {
	Name    string
	DNSName string
	Records []rrSetFixture
}

type rrSetFixture struct {
	Name    string
	Type    string
	RRDatas []string
}

var demoManagedZones = []managedZoneFixture{
	{
		Name:    "ctk-demo-zone",
		DNSName: "demo.ctk.local.",
		Records: []rrSetFixture{
			{Name: "demo.ctk.local.", Type: "A", RRDatas: []string{"203.0.113.41"}},
			{Name: "www.demo.ctk.local.", Type: "CNAME", RRDatas: []string{"demo.ctk.local."}},
		},
	},
}

func findManagedZone(name string) (managedZoneFixture, bool) {
	name = strings.TrimSpace(name)
	for _, zone := range demoManagedZones {
		if zone.Name == name {
			return zone, true
		}
	}
	return managedZoneFixture{}, false
}
