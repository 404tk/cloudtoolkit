package utils

import "time"

const (
	Provider      = "provider"
	Payload       = "payload"
	AccessKey     = "accesskey"
	SecretKey     = "secretkey"
	SecurityToken = "token"
	Region        = "region"
	ProjectID     = "projectId"
	Version       = "version"
)

const (
	AzureClientId       = "clientId"
	AzureClientSecret   = "clientSecret"
	AzureTenantId       = "tenantId"
	AzureSubscriptionId = "subscriptionId"
)

const (
	GCPserviceAccountJSON = "base64Json"
)

const (
	Metadata    = "metadata"
	BucketCheck = "list all"
	EventCheck  = "dump all"
)

var (
	DoSave       bool
	ListPolicies bool
	LogDir       string
	Cloudlist    []string
	IAMUserCheck string
	RDSAccount   string
	RunTimeout   time.Duration
)
