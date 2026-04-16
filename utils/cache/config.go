package cache

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/404tk/cloudtoolkit/utils/logger"
)

var Cfg *InitCfg

type InitCfg struct {
	Path  string
	Creds []Credential
	mu    sync.RWMutex
}

// Snapshot returns a shallow copy of Creds safe for iteration outside the package.
func (cfg *InitCfg) Snapshot() []Credential {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	out := make([]Credential, len(cfg.Creds))
	copy(out, cfg.Creds)
	return out
}

func init() {
	Cfg = NewConfig()
}

func NewConfig() *InitCfg {
	cfg := &InitCfg{}
	path := filepath.Join(userHomeDir(), ".config/cloudtoolkit/config.json")
	if v, _ := filepath.Glob(path); len(v) == 0 {
		err := os.MkdirAll(filepath.Dir(path), 0700)
		if err != nil {
			logger.Error("Could not mkdir:", err.Error())
			return cfg
		}
	} else {
		_ = os.Chmod(path, 0600)
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
	Cfg.mu.RLock()
	snapshot := make([]Credential, len(Cfg.Creds))
	copy(snapshot, Cfg.Creds)
	path := Cfg.Path
	Cfg.mu.RUnlock()

	data, err := json.MarshalIndent(snapshot, "", "\t")
	if err != nil {
		logger.Error("Failed to marshal credentials:", err.Error())
		return
	}
	err = os.WriteFile(path, data, 0600)
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
