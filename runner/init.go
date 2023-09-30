package runner

import (
	"fmt"
	"os"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/spf13/viper"
)

var filename = "config.yaml"

func InitConfig() {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) || err != nil {
		err = os.WriteFile(filename, []byte(defaultConfigFile), os.ModePerm)
	}
	viper.AddConfigPath(".")
	viper.SetConfigFile(filename)
	err = viper.ReadInConfig()
	if err != nil {
		logger.Fatalf("Read config failed: %v\n", err)
	}

	utils.DoSave = viper.GetBool("common.log_enable")
	utils.LogDir = viper.GetString("common.log_dir")
	utils.Cloudlist = viper.GetStringSlice("cloudlist")

	utils.BackdoorUser = fmt.Sprintf("%s %s %s",
		viper.GetString("backdoor-user.action"),
		viper.GetString("backdoor-user.username"),
		viper.GetString("backdoor-user.password"),
	)

	utils.DBAccount = fmt.Sprintf("%s:%s",
		viper.GetString("database-account.username"),
		viper.GetString("database-account.password"),
	)
}

const defaultConfigFile = `common:
  log_enable: true
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
