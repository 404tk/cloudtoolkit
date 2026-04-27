package catalog

import (
	"sort"
	"strings"

	"github.com/404tk/cloudtoolkit/utils"
)

var providerSpecs = map[string]ProviderSpec{
	"aws": {
		Options: []ProviderOption{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "us-east-1", Description: "N. Virginia"},
			{Text: "us-east-2", Description: "Ohio"},
			{Text: "us-west-1", Description: "N. California"},
			{Text: "us-west-2", Description: "Oregon"},
			{Text: "ap-east-1", Description: "Hong Kong"},
			{Text: "ap-southeast-1", Description: "Singapore"},
			{Text: "ap-southeast-2", Description: "Sydney"},
			{Text: "ap-northeast-1", Description: "Tokyo"},
			{Text: "ap-northeast-2", Description: "Seoul"},
			{Text: "eu-west-1", Description: "Ireland"},
			{Text: "eu-central-1", Description: "Frankfurt"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket"},
	},
	"alibaba": {
		Options: []ProviderOption{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "cn-beijing", Description: "Beijing"},
			{Text: "cn-hangzhou", Description: "Hangzhou"},
			{Text: "cn-shanghai", Description: "Shanghai"},
			{Text: "cn-shenzhen", Description: "Shenzhen"},
			{Text: "cn-hongkong", Description: "Hong Kong"},
			{Text: "ap-southeast-1", Description: "Singapore"},
			{Text: "us-east-1", Description: "Virginia"},
			{Text: "eu-central-1", Description: "Frankfurt"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket", "event", "vm", "database"},
	},
	"tencent": {
		Options: []ProviderOption{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "ap-beijing", Description: "Beijing"},
			{Text: "ap-shanghai", Description: "Shanghai"},
			{Text: "ap-guangzhou", Description: "Guangzhou"},
			{Text: "ap-hongkong", Description: "Hong Kong"},
			{Text: "ap-singapore", Description: "Singapore"},
			{Text: "ap-seoul", Description: "Seoul"},
			{Text: "ap-tokyo", Description: "Tokyo"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket", "vm"},
	},
	"huawei": {
		Options: []ProviderOption{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "cn-north-4", Description: "Beijing 4"},
			{Text: "cn-east-3", Description: "Shanghai 1"},
			{Text: "cn-south-1", Description: "Guangzhou"},
			{Text: "ap-southeast-1", Description: "Hong Kong"},
			{Text: "eu-west-101", Description: "Dublin"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket"},
	},
	"volcengine": {
		Options: []ProviderOption{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "cn-beijing", Description: "Beijing"},
			{Text: "cn-shanghai", Description: "Shanghai"},
			{Text: "ap-southeast-1", Description: "Singapore"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket", "vm"},
	},
	"jdcloud": {
		Options: []ProviderOption{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Regions: []Suggestion{
			{Text: "all", Description: "enumerate all configured regions"},
			{Text: "cn-north-1", Description: "Beijing"},
			{Text: "cn-east-2", Description: "Shanghai"},
			{Text: "cn-east-1", Description: "Suqian"},
			{Text: "cn-south-1", Description: "Guangzhou"},
		},
		Capabilities: []string{"cloudlist", "iam", "bucket", "vm"},
	},
	"ucloud": {
		Options: []ProviderOption{
			{Name: utils.AccessKey, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.SecretKey, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.SecurityToken, Description: "Security Token", Sensitive: true},
			{Name: utils.Region, Description: "Region", Default: "all"},
			{Name: utils.ProjectID, Description: "Project ID"},
		},
		Regions: []Suggestion{
			{Text: "all", Description: "enumerate all accessible regions"},
			{Text: "cn-bj2", Description: "Beijing"},
			{Text: "cn-sh2", Description: "Shanghai"},
			{Text: "cn-gd", Description: "Guangzhou"},
			{Text: "hk", Description: "Hong Kong"},
			{Text: "sg", Description: "Singapore"},
			{Text: "us-ca", Description: "Los Angeles"},
			{Text: "th-bkk", Description: "Bangkok"},
			{Text: "ge-fra", Description: "Frankfurt"},
		},
		Capabilities: []string{"cloudlist"},
	},
	"azure": {
		Options: []ProviderOption{
			{Name: utils.AzureClientId, Description: "Key ID", Required: true, Sensitive: true},
			{Name: utils.AzureClientSecret, Description: "Secret", Required: true, Sensitive: true},
			{Name: utils.AzureTenantId, Description: "Tenant ID", Required: true},
			{Name: utils.AzureSubscriptionId, Description: "Subscription ID"},
			{Name: utils.Version, Description: "International or custom edition"},
		},
		Capabilities: []string{"cloudlist"},
	},
	"gcp": {
		Options: []ProviderOption{
			{Name: utils.GCPserviceAccountJSON, Description: "GCP Credential encoded through Base64", Required: true, Sensitive: true},
		},
		Capabilities: []string{"cloudlist"},
	},
}

var optionDescriptions = buildOptionDescriptions()
var sensitiveOptions = buildSensitiveOptions()

func ProviderSpecFor(name string) (ProviderSpec, bool) {
	spec, ok := providerSpecs[strings.TrimSpace(name)]
	return spec, ok
}

func DefaultProviderConfig(name string) (map[string]string, bool) {
	spec, ok := ProviderSpecFor(name)
	if !ok {
		return nil, false
	}
	cfg := make(map[string]string, len(spec.Options))
	for _, option := range spec.Options {
		cfg[option.Name] = option.Default
	}
	return cfg, true
}

func ProviderCapabilities(name string) []string {
	spec, ok := ProviderSpecFor(name)
	if !ok {
		return nil
	}
	return append([]string(nil), spec.Capabilities...)
}

func ProviderSupportsCapability(provider, capability string) bool {
	for _, item := range ProviderCapabilities(provider) {
		if item == capability {
			return true
		}
	}
	return false
}

func ProviderRegions(name string) []Suggestion {
	spec, ok := ProviderSpecFor(name)
	if !ok {
		return nil
	}
	out := make([]Suggestion, len(spec.Regions))
	copy(out, spec.Regions)
	return out
}

func ProviderOptions(name string) []ProviderOption {
	spec, ok := ProviderSpecFor(name)
	if !ok {
		return nil
	}
	out := make([]ProviderOption, len(spec.Options))
	copy(out, spec.Options)
	return out
}

func OptionDescription(name string) string {
	return optionDescriptions[name]
}

func SensitiveOption(name string) bool {
	_, ok := sensitiveOptions[name]
	return ok
}

func OptionNames() []string {
	names := make([]string, 0, len(optionDescriptions))
	for name := range optionDescriptions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func buildOptionDescriptions() map[string]string {
	items := make(map[string]string)
	for _, spec := range providerSpecs {
		for _, option := range spec.Options {
			desc := option.Description
			switch {
			case option.Default != "":
				desc += " (Default: " + option.Default + ")"
			case !option.Required:
				desc += " (Optional)"
			}
			items[option.Name] = desc
		}
	}
	items[utils.Payload] = "Validation payload (Default: cloudlist)"
	items[utils.Metadata] = "Set the payload with additional arguments (Optional)"
	return items
}

func buildSensitiveOptions() map[string]struct{} {
	items := make(map[string]struct{})
	for _, spec := range providerSpecs {
		for _, option := range spec.Options {
			if option.Sensitive {
				items[option.Name] = struct{}{}
			}
		}
	}
	return items
}
