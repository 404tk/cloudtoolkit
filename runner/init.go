package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"gopkg.in/yaml.v3"
)

const legacyFilename = "config.yaml"

type Config struct {
	Common struct {
		LogEnable      bool   `yaml:"log_enable"`
		ListPolicies   bool   `yaml:"list_policies"`
		LogDir         string `yaml:"log_dir"`
		TimeoutMinutes int    `yaml:"timeout_minutes"`
	} `yaml:"common"`
	Cloudlist    []string `yaml:"cloudlist"`
	BackdoorUser struct {
		Action   string `yaml:"action"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"backdoor-user"`
	DatabaseAccount struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"database-account"`
}

func resolveConfigPath() string {
	if _, err := os.Stat(legacyFilename); err == nil {
		return legacyFilename
	}

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Could not resolve home directory:", err)
		return legacyFilename
	}
	path := filepath.Join(home, ".config", "cloudtoolkit", legacyFilename)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		logger.Error("Could not create config directory:", err)
		return path
	}
	if err := os.WriteFile(path, []byte(defaultConfigFile), 0600); err != nil {
		logger.Error("Could not seed default config:", err)
	}
	return path
}

func InitConfig() {
	filename := resolveConfigPath()

	data, err := os.ReadFile(filename)
	if err != nil {
		logger.Error(fmt.Sprintf("Read config failed (%s): %v — falling back to defaults", filename, err))
		data = []byte(defaultConfigFile)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Error(fmt.Sprintf("Parse config failed (%s): %v — falling back to defaults", filename, err))
		_ = yaml.Unmarshal([]byte(defaultConfigFile), &cfg)
	}

	utils.DoSave = cfg.Common.LogEnable
	utils.ListPolicies = cfg.Common.ListPolicies
	utils.LogDir = cfg.Common.LogDir
	utils.Cloudlist = cfg.Cloudlist
	if cfg.Common.TimeoutMinutes > 0 {
		utils.RunTimeout = time.Duration(cfg.Common.TimeoutMinutes) * time.Minute
	} else {
		utils.RunTimeout = 10 * time.Minute
	}

	utils.BackdoorUser = fmt.Sprintf("%s %s %s",
		cfg.BackdoorUser.Action,
		cfg.BackdoorUser.Username,
		cfg.BackdoorUser.Password,
	)

	utils.DBAccount = fmt.Sprintf("%s:%s",
		cfg.DatabaseAccount.Username,
		cfg.DatabaseAccount.Password,
	)
}

const defaultConfigFile = `common:
  log_enable: false
  list_policies: false
  log_dir: logs
  timeout_minutes: 10

cloudlist:
  - balance
  - host
  - domain
  - account
  - database
  - bucket
  - sms
  - log

backdoor-user:
  action: add
  username: ctkguest
  password: 1QAZ2wsx@Asdlkj

database-account:
  username: ctkguest
  password: 1QAZ2wsx@Asdlkj
`
