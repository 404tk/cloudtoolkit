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

func findServiceAccount(emailOrUniqueID string) (serviceAccountFixture, bool) {
	emailOrUniqueID = strings.TrimSpace(emailOrUniqueID)
	for _, sa := range demoServiceAccounts {
		if strings.EqualFold(sa.Email, emailOrUniqueID) || sa.UniqueID == emailOrUniqueID {
			return sa, true
		}
	}
	return serviceAccountFixture{}, false
}

// demoBindings is the seeded project IAM policy. The replay returns this on
// the first getIamPolicy call, then mutates it on setIamPolicy under
// transport-state.
var demoBindings = []bindingFixture{
	{Role: "roles/owner", Members: []string{"user:ctk-owner@example.com"}},
	{Role: "roles/viewer", Members: []string{
		"serviceAccount:ctk-readonly@ctk-demo-project.iam.gserviceaccount.com",
	}},
}

type bindingFixture struct {
	Role    string
	Members []string
}

// demoSAKeys is the seed list of system-managed keys returned by
// projects.serviceAccounts.keys.list. Replays add user-managed keys to
// transport-state on create.
var demoSAKeys = map[string][]saKeyFixture{
	"ctk-demo@ctk-demo-project.iam.gserviceaccount.com": {
		{KeyID: "00000000aaaaaaaa1111111122222222deadbeef", KeyType: "SYSTEM_MANAGED", ValidAfter: "2026-01-01T00:00:00Z", ValidBefore: "2027-01-01T00:00:00Z"},
	},
	"ctk-readonly@ctk-demo-project.iam.gserviceaccount.com": {
		{KeyID: "11111111bbbbbbbb2222222233333333cafef00d", KeyType: "SYSTEM_MANAGED", ValidAfter: "2026-01-01T00:00:00Z", ValidBefore: "2027-01-01T00:00:00Z"},
	},
}

type saKeyFixture struct {
	KeyID       string
	KeyType     string
	ValidAfter  string
	ValidBefore string
}

// seedSAKeys clones the seed map so the transport can mutate it freely.
func seedSAKeys() map[string][]saKeyFixture {
	out := make(map[string][]saKeyFixture, len(demoSAKeys))
	for k, v := range demoSAKeys {
		out[k] = append([]saKeyFixture(nil), v...)
	}
	return out
}

// seedBindings clones the seed slice so the transport can mutate it freely.
func seedBindings() []bindingFixture {
	out := make([]bindingFixture, len(demoBindings))
	for i, b := range demoBindings {
		out[i] = bindingFixture{Role: b.Role, Members: append([]string(nil), b.Members...)}
	}
	return out
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
