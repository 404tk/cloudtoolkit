package replay

import "strings"

type vmFixture struct {
	InstanceID       string
	Hostname         string
	Status           string
	OSType           string
	Region           string
	PrivateIPAddress string
	ElasticIPAddress string
}

var demoVMInstances = []vmFixture{
	{
		InstanceID:       "i-jdc001",
		Hostname:         "ctk-demo-bastion",
		Status:           "running",
		OSType:           "linux",
		Region:           "cn-north-1",
		PrivateIPAddress: "192.168.0.61",
		ElasticIPAddress: "203.0.113.81",
	},
	{
		InstanceID:       "i-jdc002",
		Hostname:         "ctk-demo-app",
		Status:           "running",
		OSType:           "linux",
		Region:           "cn-north-1",
		PrivateIPAddress: "192.168.0.62",
	},
	{
		InstanceID:       "i-jdc101",
		Hostname:         "ctk-demo-edge",
		Status:           "running",
		OSType:           "linux",
		Region:           "cn-east-2",
		PrivateIPAddress: "192.168.10.71",
		ElasticIPAddress: "203.0.113.82",
	},
}

func vmsForRegion(region string) []vmFixture {
	region = strings.TrimSpace(region)
	out := make([]vmFixture, 0, len(demoVMInstances))
	for _, vm := range demoVMInstances {
		if vm.Region == region {
			out = append(out, vm)
		}
	}
	return out
}

type lavmFixture struct {
	InstanceID     string
	InstanceName   string
	Status         string
	Region         string
	PublicIP       string
	PrivateIP      string
	ImageID        string
	BusinessStatus string
}

var demoLAVMInstances = []lavmFixture{
	{
		InstanceID:   "lavm-001",
		InstanceName: "ctk-demo-lavm-edge",
		Status:       "running",
		Region:       "cn-north-1",
		PublicIP:     "203.0.113.91",
		PrivateIP:    "192.168.20.11",
		ImageID:      "img-lavm-linux",
	},
}

func lavmForRegion(region string) []lavmFixture {
	region = strings.TrimSpace(region)
	out := make([]lavmFixture, 0, len(demoLAVMInstances))
	for _, item := range demoLAVMInstances {
		if item.Region == region {
			out = append(out, item)
		}
	}
	return out
}

type subUserFixture struct {
	Pin        string
	Name       string
	Account    string
	CreateTime string
}

var demoBaseSubUsers = []subUserFixture{
	{
		Pin:        "ctk-demo-master:ctk-demo-readonly",
		Name:       "ctk-demo-readonly",
		Account:    "ctk-demo-readonly@" + demoMasterPin,
		CreateTime: "2026-04-20T12:00:00Z",
	},
	{
		Pin:        "ctk-demo-master:ctk-demo-bot",
		Name:       "ctk-demo-bot",
		Account:    "ctk-demo-bot@" + demoMasterPin,
		CreateTime: "2026-04-21T15:30:00Z",
	},
}

type bucketFixture struct {
	Name string
}

var demoBuckets = []bucketFixture{
	{Name: "ctk-validation-logs"},
	{Name: "ctk-validation-archive"},
}
