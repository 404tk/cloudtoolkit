package console

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

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
				}
			}
		}
	}
	fmt.Println("Usage of sessions:\n\t-i, internation [id]\n\t-k, kill [id]")
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
