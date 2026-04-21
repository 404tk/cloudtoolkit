package console

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/404tk/cloudtoolkit/pkg/providers"
	"github.com/404tk/cloudtoolkit/runner/payloads"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
)

type credential struct {
	Id        int    `table:"ID"`
	User      string `table:"User"`
	AccessKey string `table:"AccessKey"`
	Provider  string `table:"Provider"`
	Note      string `table:"Note"`
}

var creds []credential
var cred_ids map[int]string

func init() {
	loadCred()
}

func sessions(args []string) {
	if len(args) == 0 {
		listSessions()
		return
	} else if len(args) == 2 {
		if args[0] == "-c" {
			if args[1] == "all" {
				checkCred("all")
				return
			}
		}
		uuid := getUuid(args[1])
		if len(uuid) > 0 {
			switch args[0] {
			case "-i":
				internation(uuid)
				return
			case "-k":
				cache.Cfg.CredDelete(uuid)
				loadCred()
				return
			case "-c":
				checkCred(uuid)
				return
			}
		}
	} else if len(args) == 1 && args[0] == "-c" {
		checkCred("all")
		return
	}
	fmt.Println("Usage of sessions:\n\t-i, interact [id]\n\t-k, kill [id]\n\t-c, check [id|all]")
}

func note(args []string) {
	if len(args) < 2 {
		fmt.Println("Example: note 1 Test")
		return
	}
	uuid := getUuid(args[0])
	if len(uuid) > 0 {
		cache.Cfg.CredNote(uuid, args[1])
	}
}

func getUuid(s string) string {
	sid, err := strconv.Atoi(s)
	if err == nil {
		if uuid, ok := cred_ids[sid]; ok {
			return uuid
		}
	}
	return ""
}

func loadCred() {
	creds = []credential{}
	cred_ids = make(map[int]string)
	for i, v := range cache.Cfg.Snapshot() {
		creds = append(creds, credential{
			Id:        i + 1,
			User:      v.User,
			AccessKey: v.AccessKey,
			Provider:  v.Provider,
			Note:      v.Note,
		})
		cred_ids[i+1] = v.UUID
	}
}

func listSessions() {
	loadCred()
	table.Output(creds)
}

func internation(uuid string) {
	m, ok := decodeSessionConfig(cache.Cfg.CredSelect(uuid))
	if !ok {
		return
	}
	if provider, ok := m[utils.Provider]; ok {
		config = m
		if _, ok := config[utils.Metadata]; !ok {
			config[utils.Metadata] = ""
		}
		if name, ok := config[utils.Payload]; ok {
			config[utils.Payload] = payloads.ResolveName(name)
		}
		if target := shellTargetFromConfig(config); target != "" {
			rememberShellTarget(target, provider, "cached session")
		}
		startProviderConsole(provider)
	}
}

func checkCred(uuid string) {
	for _, cred := range cache.Cfg.Snapshot() {
		if uuid != "all" && cred.UUID != uuid {
			continue
		}
		m, ok := decodeSessionConfig(cred.JsonData)
		if !ok {
			continue
		}
		if value, ok := m[utils.Provider]; ok {
			if !providers.Supports(value) {
				continue
			}
			m[utils.Payload] = "cloudlist"
			_, err := providers.New(value, m)
			if err != nil {
				logger.Error(fmt.Sprintf("%s(%s) check failed.", cred.User, cred.AccessKey))
			}
		}
	}
}

func decodeSessionConfig(data string) (map[string]string, bool) {
	m := make(map[string]string)
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		logger.Error("Unmarshal failed:", err.Error())
		return nil, false
	}
	return m, true
}
