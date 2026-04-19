package auth

import (
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Credential is the provider-local Azure credential shape used by the
// lightweight ARM client. SubscriptionID is optional; when empty the provider
// enumerates all visible subscriptions first.
type Credential struct {
	ClientID       string
	ClientSecret   string
	TenantID       string
	SubscriptionID string
	Cloud          Cloud
}

type Cloud string

const (
	CloudPublic  Cloud = "public"
	CloudChina   Cloud = "china"
	CloudUSGov   Cloud = "usgov"
	CloudGermany Cloud = "germany"
)

func New(clientID, clientSecret, tenantID, subscriptionID string, cloud Cloud) Credential {
	return Credential{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		Cloud:          normalizeCloud(cloud),
	}
}

func FromOptions(options schema.Options) (Credential, error) {
	clientID, ok := options.GetMetadata(utils.AzureClientId)
	if !ok {
		return Credential{}, &schema.ErrNoSuchKey{Name: utils.AzureClientId}
	}
	clientSecret, ok := options.GetMetadata(utils.AzureClientSecret)
	if !ok {
		return Credential{}, &schema.ErrNoSuchKey{Name: utils.AzureClientSecret}
	}
	tenantID, ok := options.GetMetadata(utils.AzureTenantId)
	if !ok {
		return Credential{}, &schema.ErrNoSuchKey{Name: utils.AzureTenantId}
	}
	subscriptionID, _ := options.GetMetadata(utils.AzureSubscriptionId)
	version, _ := options.GetMetadata(utils.Version)
	return New(clientID, clientSecret, tenantID, subscriptionID, cloudFromVersion(version)), nil
}

func (c Credential) Validate() error {
	switch {
	case strings.TrimSpace(c.ClientID) == "":
		return errors.New("azure credential: empty client id")
	case strings.TrimSpace(c.ClientSecret) == "":
		return errors.New("azure credential: empty client secret")
	case strings.TrimSpace(c.TenantID) == "":
		return errors.New("azure credential: empty tenant id")
	default:
		return nil
	}
}

func (c Cloud) ActiveDirectoryEndpoint() string {
	switch normalizeCloud(c) {
	case CloudChina:
		return "https://login.chinacloudapi.cn/"
	case CloudUSGov:
		return "https://login.microsoftonline.us/"
	case CloudGermany:
		return "https://login.microsoftonline.de/"
	default:
		return "https://login.microsoftonline.com/"
	}
}

func (c Cloud) ResourceManagerEndpoint() string {
	switch normalizeCloud(c) {
	case CloudChina:
		return "https://management.chinacloudapi.cn/"
	case CloudUSGov:
		return "https://management.usgovcloudapi.net/"
	case CloudGermany:
		return "https://management.microsoftazure.de/"
	default:
		return "https://management.azure.com/"
	}
}

func cloudFromVersion(version string) Cloud {
	switch strings.TrimSpace(strings.ToLower(version)) {
	case "", "public":
		return CloudPublic
	case "china":
		return CloudChina
	case "usgov":
		return CloudUSGov
	case "germany":
		return CloudGermany
	default:
		return CloudPublic
	}
}

func normalizeCloud(cloud Cloud) Cloud {
	switch Cloud(strings.ToLower(strings.TrimSpace(string(cloud)))) {
	case CloudChina:
		return CloudChina
	case CloudUSGov:
		return CloudUSGov
	case CloudGermany:
		return CloudGermany
	default:
		return CloudPublic
	}
}
