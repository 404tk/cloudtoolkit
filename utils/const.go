package utils

const (
	Provider      = "provider"
	Payload       = "payload"
	AccessKey     = "accesskey"
	SecretKey     = "secretkey"
	SecurityToken = "token"
	Region        = "region"
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
	Metadata   = "metadata"
	BucketDump = "list all"
	EventDump  = "dump all"
)

var (
	DoSave       bool
	ListPolicies bool
	LogDir       string
	Cloudlist    []string
	BackdoorUser string
	DBAccount    string
)
