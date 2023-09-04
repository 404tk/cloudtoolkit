package cache

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"

	"github.com/404tk/cloudtoolkit/utils/logger"
)

var Cfg *InitCfg

type InitCfg struct {
	Path  string
	Creds []Credential
}

func init() {
	Cfg = NewConfig()
}

func NewConfig() *InitCfg {
	cfg := &InitCfg{}
	path := filepath.Join(userHomeDir(), ".config/cloudtoolkit/config.json")
	if v, _ := filepath.Glob(path); len(v) == 0 {
		err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err != nil {
			logger.Error("Could not mkdir:", err.Error())
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
		return
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&creds)
	if err != nil {
		logger.Error("Get credential info failed:", err.Error())
		return
	}

	return
}

func SaveFile() {
	data, err := json.Marshal(Cfg.Creds)
	err = os.WriteFile(Cfg.Path, data, 0644)
	if err != nil {
		logger.Error("Failed to write the config file:", err.Error())
	}

}

func userHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		logger.Error("Could not get user home directory:", err)
		return ""
	}
	return usr.HomeDir
}
