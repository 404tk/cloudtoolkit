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

type userValidationConfig struct {
	Action   string `yaml:"action"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type databaseAccountConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Config struct {
	Common struct {
		LogEnable      bool   `yaml:"log_enable"`
		ListPolicies   bool   `yaml:"list_policies"`
		LogDir         string `yaml:"log_dir"`
		TimeoutMinutes int    `yaml:"timeout_minutes"`
	} `yaml:"common"`
	Cloudlist                 []string              `yaml:"cloudlist"`
	IAMUserCheck              userValidationConfig  `yaml:"iam-user-check"`
	LegacyIAMUserValidation   userValidationConfig  `yaml:"iam-user-validation"`
	LegacyBackdoorUser        userValidationConfig  `yaml:"backdoor-user"`
	RDSAccountCheck           databaseAccountConfig `yaml:"rds-account-check"`
	LegacyDBAccountValidation databaseAccountConfig `yaml:"database-account-validation"`
	LegacyDatabaseAccount     databaseAccountConfig `yaml:"database-account"`
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

	iamUserCheck := cfg.IAMUserCheck
	if iamUserCheck == (userValidationConfig{}) {
		iamUserCheck = cfg.LegacyIAMUserValidation
	}
	if iamUserCheck == (userValidationConfig{}) {
		iamUserCheck = cfg.LegacyBackdoorUser
	}
	utils.IAMUserCheck = fmt.Sprintf("%s %s %s",
		iamUserCheck.Action,
		iamUserCheck.Username,
		iamUserCheck.Password,
	)

	rdsAccountCheck := cfg.RDSAccountCheck
	if rdsAccountCheck == (databaseAccountConfig{}) {
		rdsAccountCheck = cfg.LegacyDBAccountValidation
	}
	if rdsAccountCheck == (databaseAccountConfig{}) {
		rdsAccountCheck = cfg.LegacyDatabaseAccount
	}
	utils.RDSAccount = fmt.Sprintf("%s:%s",
		rdsAccountCheck.Username,
		rdsAccountCheck.Password,
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

iam-user-check:
  action: add
  username: ctkguest
  password: 1QAZ2wsx@Asdlkj

rds-account-check:
  username: ctkguest
  password: 1QAZ2wsx@Asdlkj
`
