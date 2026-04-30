package replay

import (
	"strings"
	"sync"
)

type State struct {
	Active   bool
	Provider string
}

type Credentials struct {
	AccessKey string
	SecretKey string
	// Extras carry additional option-key / value pairs needed to fully
	// configure providers that do not fit the AccessKey + SecretKey shape
	// (e.g. azure tenantId / subscriptionId, gcp base64Json). They are
	// surfaced in the demo replay banner alongside AccessKey / SecretKey.
	Extras []NamedValue
}

// NamedValue is a key/value pair used by demo replay banners and
// option-fill helpers to advertise extra credential fields.
type NamedValue struct {
	Name  string
	Value string
}

type providerMeta struct {
	Credentials Credentials
	Payloads    []string
}

var (
	stateMu sync.RWMutex
	state   State
)

var providers = map[string]providerMeta{
	"alibaba": {
		Credentials: Credentials{
			AccessKey: "LTAI4tDVhjxvrWKTsEXAMPLE",
			SecretKey: "EXAMPLEv2fYAa2s7GhvLun7xqctKEY",
		},
		Payloads: []string{
			"cloudlist",
			"iam-user-check",
			"bucket-check",
			"event-check",
			"rds-account-check",
			"instance-cmd-check",
		},
	},
	"volcengine": {
		Credentials: Credentials{
			AccessKey: "AKLTOTgzY2UyYzA1NDQ5NGE5MzkEXAMPLE",
			SecretKey: "QkN5ZlJmM0d1R2JMb2M5dVhLQXBQd1pHc2ZEXAMPLE",
		},
		Payloads: []string{
			"cloudlist",
			"iam-user-check",
			"bucket-check",
			"instance-cmd-check",
		},
	},
	"tencent": {
		Credentials: Credentials{
			AccessKey: "AKIDz8krbsJ5yKBZQpn74WFkmLPx3EXAMPLE",
			SecretKey: "Gu5t9xGARNpq86cd98joQYCN3EXAMPLE",
		},
		Payloads: []string{
			"cloudlist",
			"iam-user-check",
			"bucket-check",
			"instance-cmd-check",
		},
	},
	"aws": {
		Credentials: Credentials{
			AccessKey: "AKIAIOSFODNN7EXAMPLE",
			SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
		Payloads: []string{
			"cloudlist",
			"iam-user-check",
			"bucket-check",
		},
	},
	"huawei": {
		Credentials: Credentials{
			AccessKey: "HWEXAMPLEAKIDz8krbsJ5y",
			SecretKey: "HwSkExampleGu5t9xGARNpq86cd98joQYCN3EXAMPLE",
		},
		Payloads: []string{
			"cloudlist",
			"iam-user-check",
			"bucket-check",
		},
	},
	"azure": {
		Credentials: Credentials{
			AccessKey: "11111111-2222-3333-4444-555555555555",
			SecretKey: "AzExampleClientSecretEXAMPLEvalueDEMO0000",
			Extras: []NamedValue{
				{Name: "clientId", Value: "11111111-2222-3333-4444-555555555555"},
				{Name: "clientSecret", Value: "AzExampleClientSecretEXAMPLEvalueDEMO0000"},
				{Name: "tenantId", Value: "11111111-2222-3333-4444-555555555555"},
				{Name: "subscriptionId", Value: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
			},
		},
		Payloads: []string{
			"cloudlist",
			"role-binding-check",
			"bucket-acl-check",
		},
	},
	"gcp": {
		Credentials: Credentials{
			AccessKey: "ctk-demo-project",
			SecretKey: "(see base64Json below)",
			Extras: []NamedValue{
				{Name: "base64Json", Value: gcpDemoServiceAccountJSON},
			},
		},
		Payloads: []string{
			"cloudlist",
			"role-binding-check",
			"sa-key-check",
		},
	},
	"jdcloud": {
		Credentials: Credentials{
			AccessKey: "JDC_AKLTEXAMPLE000000000001",
			SecretKey: "JDCExampleSecretKeyValueDEMOreplay00000",
		},
		Payloads: []string{
			"cloudlist",
			"iam-user-check",
			"bucket-check",
		},
	},
	"ucloud": {
		Credentials: Credentials{
			AccessKey: "ucloudpubkey-EXAMPLE-ctkdemo-replay-public",
			SecretKey: "ucloudprivkey-EXAMPLE-ctkdemo-replay-secret",
		},
		Payloads: []string{
			"cloudlist",
			"iam-user-check",
		},
	},
}

func Enable(provider string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	state = State{
		Active:   true,
		Provider: normalizeProvider(provider),
	}
}

func Disable() {
	stateMu.Lock()
	defer stateMu.Unlock()
	state = State{}
}

func IsActive() bool {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return state.Active && state.Provider != ""
}

func ActiveProvider() string {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return state.Provider
}

func IsActiveForProvider(provider string) bool {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return state.Active && state.Provider != "" && state.Provider == normalizeProvider(provider)
}

func SupportsProvider(provider string) bool {
	_, ok := providers[normalizeProvider(provider)]
	return ok
}

func CredentialsFor(provider string) (Credentials, bool) {
	meta, ok := providers[normalizeProvider(provider)]
	if !ok {
		return Credentials{}, false
	}
	return meta.Credentials, true
}

func SupportedPayloads(provider string) []string {
	meta, ok := providers[normalizeProvider(provider)]
	if !ok || len(meta.Payloads) == 0 {
		return nil
	}
	return append([]string(nil), meta.Payloads...)
}

func normalizeProvider(provider string) string {
	return strings.TrimSpace(provider)
}
