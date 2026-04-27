package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
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
		LogFormat      string `yaml:"log_format"`
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

func firstUserValidationConfig(values ...userValidationConfig) userValidationConfig {
	for _, value := range values {
		if value != (userValidationConfig{}) {
			return value
		}
	}
	return userValidationConfig{}
}

func firstDatabaseAccountConfig(values ...databaseAccountConfig) databaseAccountConfig {
	for _, value := range values {
		if value != (databaseAccountConfig{}) {
			return value
		}
	}
	return databaseAccountConfig{}
}

// InitConfig parses config.yaml (CWD or XDG) and returns the resulting *env.Env.
// As a side effect it pins the same env via env.SetActive so capability methods
// without ctx (EventDump, parseRDSAccount) can fall back to env.Active().
//
// cmd/main.go calls this once at startup and threads the returned env through
// to the REPL or headless dispatcher.
func InitConfig() *env.Env {
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

	e := &env.Env{
		LogEnable:    cfg.Common.LogEnable,
		ListPolicies: cfg.Common.ListPolicies,
		LogDir:       cfg.Common.LogDir,
		Cloudlist:    append([]string(nil), cfg.Cloudlist...),
	}
	if cfg.Common.TimeoutMinutes > 0 {
		e.RunTimeout = time.Duration(cfg.Common.TimeoutMinutes) * time.Minute
	} else {
		e.RunTimeout = 10 * time.Minute
	}
	logger.SetFormat(logger.Format(strings.ToLower(strings.TrimSpace(cfg.Common.LogFormat))))

	iamUserCheck := firstUserValidationConfig(
		cfg.IAMUserCheck,
		cfg.LegacyIAMUserValidation,
		cfg.LegacyBackdoorUser,
	)
	e.IAMUserCheck = fmt.Sprintf("%s %s %s",
		iamUserCheck.Action,
		iamUserCheck.Username,
		iamUserCheck.Password,
	)

	rdsAccountCheck := firstDatabaseAccountConfig(
		cfg.RDSAccountCheck,
		cfg.LegacyDBAccountValidation,
		cfg.LegacyDatabaseAccount,
	)
	e.RDSAccount = fmt.Sprintf("%s:%s",
		rdsAccountCheck.Username,
		rdsAccountCheck.Password,
	)

	env.SetActive(e)
	return e
}

const defaultConfigFile = `common:
  log_enable: false
  list_policies: false
  log_dir: logs
  timeout_minutes: 10
  log_format: text  # text | json — json emits one JSON Line per record for SIEM ingestion

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
