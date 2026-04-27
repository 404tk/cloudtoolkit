package cache

import (
	"encoding/json"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type CredentialKeyer interface {
	CredentialKey(opts map[string]string) string
}

type Credential struct {
	UUID      string
	User      string
	AccessKey string
	Provider  string
	JsonData  string
	Note      string
}

func (cfg *InitCfg) CredInsert(user string, provider any, data map[string]string) {
	cfg.ensureLoaded()
	providerName := data[utils.Provider]
	accessKey := credentialKey(provider, data)
	uuid := utils.Md5Encode(accessKey + providerName)

	b, err := json.Marshal(data)
	if err != nil {
		logger.Error("Map to json failed:", err.Error())
		return
	}

	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	for k, v := range cfg.Creds {
		if v.UUID == uuid {
			cfg.Creds[k].User = truncateString(user, 20)
			cfg.Creds[k].AccessKey = truncateString(accessKey, 35)
			cfg.Creds[k].Provider = providerName
			cfg.Creds[k].JsonData = string(b)
			return
		}
	}
	cfg.Creds = append(cfg.Creds, Credential{
		UUID:      uuid,
		User:      truncateString(user, 20),
		AccessKey: truncateString(accessKey, 35),
		Provider:  providerName,
		JsonData:  string(b),
	})
}

func credentialKey(provider any, data map[string]string) string {
	if keyer, ok := provider.(CredentialKeyer); ok {
		return keyer.CredentialKey(data)
	}
	return data[utils.AccessKey]
}

func (cfg *InitCfg) CredSelect(uuid string) string {
	cfg.ensureLoaded()
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	for _, v := range cfg.Creds {
		if v.UUID == uuid {
			return v.JsonData
		}
	}
	return ""
}

func (cfg *InitCfg) CredUpdate(uuid, data string) {
	cfg.ensureLoaded()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	for k, v := range cfg.Creds {
		if v.UUID == uuid {
			cfg.Creds[k].JsonData = data
			return
		}
	}
}

func (cfg *InitCfg) CredNote(uuid, data string) {
	cfg.ensureLoaded()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	for k, v := range cfg.Creds {
		if v.UUID == uuid {
			cfg.Creds[k].Note = data
			return
		}
	}
}

func (cfg *InitCfg) CredDelete(uuid string) {
	cfg.ensureLoaded()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	for index, v := range cfg.Creds {
		if v.UUID == uuid {
			if index == len(cfg.Creds)-1 {
				cfg.Creds = cfg.Creds[:index]
			} else {
				cfg.Creds = append(cfg.Creds[:index], cfg.Creds[index+1:]...)
			}
			return
		}
	}
}

func truncateString(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
