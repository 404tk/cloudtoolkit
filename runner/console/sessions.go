package console

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/404tk/cloudtoolkit/pkg/plugins"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/c-bata/go-prompt"
	"github.com/modood/table"
)

type credential struct {
	Id        int    `table:"ID"`
	User      string `table:"User"`
	AccessKey string `table:"AccessKey"`
	Provider  string `table:"Provider"`
}

var creds []credential
var cred_ids map[int]string

func init() {
	loadCred()
}

func sessions(args []string) {
	if len(args) == 0 {
		loadCred()
		table.Output(creds)
		return
	} else if len(args) == 2 {
		if id, err := strconv.Atoi(args[1]); err == nil {
			if uuid, ok := cred_ids[id]; ok {
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
		}
	} else if len(args) == 1 && args[0] == "-c" {
		checkCred("all")
		return
	}
	fmt.Println("Usage of sessions:\n\t-i, internation [id]\n\t-k, kill [id]\n\t-c, check all")
}

func loadCred() {
	creds = []credential{}
	cred_ids = make(map[int]string)
	for i, v := range cache.Cfg.Creds {
		creds = append(creds, credential{
			Id:        i + 1,
			User:      v.User,
			AccessKey: v.AccessKey,
			Provider:  v.Provider,
		})
		cred_ids[i+1] = v.UUID
	}
}

func internation(uuid string) {
	data := cache.Cfg.CredSelect(uuid)
	m := make(map[string]string)
	err := json.Unmarshal([]byte(data), &m)
	if err != nil {
		log.Println("[-] Unmarshal failed:", err.Error())
	}
	if provider, ok := m[utils.Provider]; ok {
		config = m
		p := prompt.New(
			Executor,
			actionCompleter,
			prompt.OptionPrefix(fmt.Sprintf("ctk > %s > ", provider)),
			prompt.OptionInputTextColor(prompt.White),
		)
		p.Run()
	}
}

func checkCred(uuid string) {
	for _, cred := range cache.Cfg.Creds {
		if uuid != "all" && cred.UUID != uuid {
			continue
		}
		m := make(map[string]string)
		err := json.Unmarshal([]byte(cred.JsonData), &m)
		if err != nil {
			log.Println("[-] Unmarshal failed:", err.Error())
		}
		if value, ok := m[utils.Provider]; ok {
			if v, ok := plugins.Providers[value]; ok {
				_, err = v.Check(m)
				if err != nil {
					log.Printf("[-] %s(%s) check failed.\n", cred.User, cred.AccessKey)
				}
			}
		}
	}
}
