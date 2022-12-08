package cache

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"sync"
)

var Cfg *InitCfg
var once sync.Once

type InitCfg struct {
	Path  string
	Creds []Credential
}

func init() {
	once.Do(func() {
		Cfg = NewConfig()
	})
}

func NewConfig() *InitCfg {
	cfg := &InitCfg{}
	path := filepath.Join(userHomeDir(), ".config/cloudtoolkit/config.json")
	if v, _ := filepath.Glob(path); len(v) == 0 {
		err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err != nil {
			log.Println("[-] Could not mkdir:", err.Error())
			return cfg
		}
	}
	cfg.Path = path
	cfg.Creds = getCreds(path)
	return cfg
}

func getCreds(path string) (creds []Credential) {
	file, err := os.Open(path)
	if err != nil {
		log.Println("[-] Get cache file failed:", err.Error())
		return
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&creds)
	if err != nil {
		log.Println("[-] Get credential info failed:", err.Error())
		return
	}

	return
}

func SaveFile() {
	data, err := json.Marshal(Cfg.Creds)
	err = ioutil.WriteFile(Cfg.Path, data, 0644)
	if err != nil {
		log.Println("[-] Failed to write the config file:", err.Error())
	}

}

func userHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Println("[-] Could not get user home directory:", err)
		return ""
	}
	return usr.HomeDir
}
