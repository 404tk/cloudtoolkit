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
	once  sync.Once
}

// Snapshot returns a shallow copy of Creds safe for iteration outside the package.
func (cfg *InitCfg) Snapshot() []Credential {
	cfg.ensureLoaded()
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	out := make([]Credential, len(cfg.Creds))
	copy(out, cfg.Creds)
	return out
}

func init() {
	Cfg = &InitCfg{}
}

func NewConfig() *InitCfg {
	cfg := &InitCfg{}
	cfg.ensureLoaded()
	return cfg
}

func (cfg *InitCfg) ensureLoaded() {
	cfg.once.Do(func() {
		path := filepath.Join(userHomeDir(), ".config/cloudtoolkit/config.json")
		cfg.Path = path
		cfg.Creds = getCreds(path)
		if _, err := os.Stat(path); err == nil {
			_ = os.Chmod(path, 0600)
		}
	})
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
	Cfg.ensureLoaded()
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
	err = os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		logger.Error("Could not mkdir:", err.Error())
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
