package replay

import (
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
)

const (
	demoAccountID    = "123456789012"
	demoAccountAlias = "ctk-validation"
)

var demoRegions = []string{
	"us-east-1",
	"us-west-2",
	"ap-southeast-1",
	"eu-west-1",
}

type ec2HostFixture struct {
	Region        string
	InstanceID    string
	PublicIP      string
	PrivateIP     string
	PublicDNSName string
	State         string
	Tags          []api.EC2Tag
}

var demoEC2Hosts = []ec2HostFixture{
	{
		Region:        "us-east-1",
		InstanceID:    "i-0a1b2c3d4e5f60001",
		PublicIP:      "203.0.113.10",
		PrivateIP:     "10.0.1.10",
		PublicDNSName: "ec2-203-0-113-10.compute-1.amazonaws.com",
		State:         "running",
		Tags: []api.EC2Tag{
			{Key: "Name", Value: "ctk-demo-bastion"},
			{Key: "env", Value: "validation"},
		},
	},
	{
		Region:     "us-east-1",
		InstanceID: "i-0a1b2c3d4e5f60002",
		PrivateIP:  "10.0.2.20",
		State:      "running",
		Tags: []api.EC2Tag{
			{Key: "Name", Value: "ctk-demo-app"},
		},
	},
	{
		Region:        "us-west-2",
		InstanceID:    "i-0a1b2c3d4e5f60101",
		PublicIP:      "198.51.100.21",
		PrivateIP:     "10.1.1.21",
		PublicDNSName: "ec2-198-51-100-21.us-west-2.compute.amazonaws.com",
		State:         "running",
		Tags: []api.EC2Tag{
			{Key: "Name", Value: "ctk-demo-edge"},
		},
	},
	{
		Region:     "ap-southeast-1",
		InstanceID: "i-0a1b2c3d4e5f60201",
		PrivateIP:  "10.2.1.31",
		State:      "stopped",
		Tags: []api.EC2Tag{
			{Key: "Name", Value: "ctk-demo-batch"},
		},
	},
	{
		Region:        "eu-west-1",
		InstanceID:    "i-0a1b2c3d4e5f60301",
		PublicIP:      "192.0.2.41",
		PrivateIP:     "10.3.1.41",
		PublicDNSName: "ec2-192-0-2-41.eu-west-1.compute.amazonaws.com",
		State:         "running",
		Tags: []api.EC2Tag{
			{Key: "Name", Value: "ctk-demo-api"},
			{Key: "team", Value: "platform"},
		},
	},
}

type iamPolicyFixture struct {
	Name string
	Arn  string
}

type iamUserFixture struct {
	UserName       string
	UserID         string
	Arn            string
	CreateDate     string
	LastLoginDate  string
	HasLogin       bool
	AttachedPolicy []iamPolicyFixture
}

var demoIAMUsers = []iamUserFixture{
	{
		UserName:      "ctk-demo-admin",
		UserID:        "AIDAIOSFODNN7EXAMPLE001",
		Arn:           "arn:aws:iam::" + demoAccountID + ":user/ctk-demo-admin",
		CreateDate:    "2026-01-12T08:00:00Z",
		LastLoginDate: "2026-04-22T09:21:00Z",
		HasLogin:      true,
		AttachedPolicy: []iamPolicyFixture{
			{Name: "AdministratorAccess", Arn: "arn:aws:iam::aws:policy/AdministratorAccess"},
		},
	},
	{
		UserName:      "ctk-demo-readonly",
		UserID:        "AIDAIOSFODNN7EXAMPLE002",
		Arn:           "arn:aws:iam::" + demoAccountID + ":user/ctk-demo-readonly",
		CreateDate:    "2026-02-04T10:30:00Z",
		LastLoginDate: "2026-04-20T15:00:00Z",
		HasLogin:      true,
		AttachedPolicy: []iamPolicyFixture{
			{Name: "ReadOnlyAccess", Arn: "arn:aws:iam::aws:policy/ReadOnlyAccess"},
		},
	},
	{
		UserName:   "ctk-demo-bot",
		UserID:     "AIDAIOSFODNN7EXAMPLE003",
		Arn:        "arn:aws:iam::" + demoAccountID + ":user/ctk-demo-bot",
		CreateDate: "2026-03-01T00:00:00Z",
		HasLogin:   false,
		AttachedPolicy: []iamPolicyFixture{
			{Name: "AmazonS3ReadOnlyAccess", Arn: "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"},
		},
	},
}

func findIAMUser(userName string) (iamUserFixture, bool) {
	userName = strings.TrimSpace(userName)
	for _, user := range demoIAMUsers {
		if user.UserName == userName {
			return user, true
		}
	}
	return iamUserFixture{}, false
}

type bucketObjectFixture struct {
	Key          string
	Size         int64
	LastModified string
	StorageClass string
}

type s3BucketFixture struct {
	Name    string
	Region  string
	Objects []bucketObjectFixture
}

var demoS3Buckets = []s3BucketFixture{
	{
		Name:   "ctk-validation-logs",
		Region: "us-east-1",
		Objects: []bucketObjectFixture{
			{Key: "audit/2026-04-20.log", Size: 12480, LastModified: "2026-04-20T23:59:00.000Z", StorageClass: "STANDARD"},
			{Key: "audit/2026-04-21.log", Size: 13950, LastModified: "2026-04-21T23:59:00.000Z", StorageClass: "STANDARD"},
			{Key: "audit/2026-04-22.log", Size: 11200, LastModified: "2026-04-22T23:59:00.000Z", StorageClass: "STANDARD"},
		},
	},
	{
		Name:   "ctk-validation-public",
		Region: "us-west-2",
		Objects: []bucketObjectFixture{
			{Key: "release/notes.md", Size: 4096, LastModified: "2026-03-12T10:00:00.000Z", StorageClass: "STANDARD"},
			{Key: "release/changelog.txt", Size: 8192, LastModified: "2026-04-15T18:30:00.000Z", StorageClass: "STANDARD"},
		},
	},
	{
		Name:   "ctk-validation-archive",
		Region: "eu-west-1",
		Objects: []bucketObjectFixture{
			{Key: "archive/2026Q1.tar.gz", Size: 1048576, LastModified: "2026-04-01T03:00:00.000Z", StorageClass: "GLACIER"},
		},
	},
}

func findS3Bucket(name string) (s3BucketFixture, bool) {
	name = strings.TrimSpace(name)
	for _, bucket := range demoS3Buckets {
		if bucket.Name == name {
			return bucket, true
		}
	}
	return s3BucketFixture{}, false
}

func ec2HostsForRegion(region string) []ec2HostFixture {
	region = strings.TrimSpace(region)
	out := make([]ec2HostFixture, 0)
	for _, host := range demoEC2Hosts {
		if host.Region == region {
			out = append(out, host)
		}
	}
	return out
}

func demoCallerArn() string {
	return "arn:aws:iam::" + demoAccountID + ":user/ctk-demo-admin"
}

func demoCallerUserID() string {
	return "AIDAIOSFODNN7EXAMPLE000"
}
