package runner

import (
	"fmt"
	"os"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"gopkg.in/yaml.v3"
)

var filename = "config.yaml"

type Config struct {
	Common struct {
		LogEnable    bool   `yaml:"log_enable"`
		ListPolicies bool   `yaml:"list_policies"`
		LogDir       string `yaml:"log_dir"`
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

func InitConfig() {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) || err != nil {
		_ = os.WriteFile(filename, []byte(defaultConfigFile), os.ModePerm)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		logger.Fatalf("Read config failed: %v\n", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Fatalf("Parse config failed: %v\n", err)
	}

	utils.DoSave = cfg.Common.LogEnable
	utils.ListPolicies = cfg.Common.ListPolicies
	utils.LogDir = cfg.Common.LogDir
	utils.Cloudlist = cfg.Cloudlist

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
