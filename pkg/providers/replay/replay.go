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
