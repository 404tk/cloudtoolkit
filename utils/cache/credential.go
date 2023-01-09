package cache

import (
	"encoding/base64"
	"encoding/json"
	"log"

	"github.com/404tk/cloudtoolkit/utils"
)

type Credential struct {
	UUID      string
	User      string
	AccessKey string
	Provider  string
	JsonData  string
}

func (cfg *InitCfg) CredInsert(user string, data map[string]string) {
	provider, _ := data[utils.Provider]
	accessKey, _ := data[utils.AccessKey]
	switch provider {
	case "azure":
		accessKey, _ = data[utils.AzureClientId]
	case "gcp":
		tojson, _ := base64.StdEncoding.DecodeString(data[utils.GCPserviceAccountJSON])
		accessKey = utils.Md5Encode(string(tojson))
	}
	uuid := utils.Md5Encode(accessKey + provider)

	b, err := json.Marshal(data)
	if err != nil {
		log.Println("[-] Map to json failed:", err.Error())
		return
	}

	if Cfg.CredSelect(uuid) != "" {
		Cfg.CredUpdate(uuid, string(b))
	} else {
		cfg.Creds = append(cfg.Creds, Credential{
			UUID:      uuid,
			User:      user,
			AccessKey: accessKey,
			Provider:  provider,
			JsonData:  string(b),
		})
	}
}

func (cfg *InitCfg) CredSelect(uuid string) string {
	for _, v := range cfg.Creds {
		if v.UUID == uuid {
			return v.JsonData
		}
	}
	return ""
}

func (cfg *InitCfg) CredUpdate(uuid, data string) {
	for k, v := range cfg.Creds {
		if v.UUID == uuid {
			cfg.Creds[k].JsonData = data
			return
		}
	}
}

func (cfg *InitCfg) CredDelete(uuid string) {
	for index, v := range cfg.Creds {
		if v.UUID == uuid {
			if index == len(cfg.Creds) {
				cfg.Creds = cfg.Creds[:index]
			} else {
				cfg.Creds = append(cfg.Creds[:index], cfg.Creds[index+1:]...)
			}
			return
		}
	}
}
