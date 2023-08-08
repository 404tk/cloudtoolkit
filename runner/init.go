package runner

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/spf13/viper"
)

var filename = "config.yaml"

func InitConfig() {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) || err != nil {
		err = ioutil.WriteFile(filename, []byte(defaultConfigFile), os.ModePerm)
	}
	viper.AddConfigPath(".")
	viper.SetConfigFile(filename)
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatalf("[-] Read config failed: %v", err)
	}

	utils.DoSave = viper.GetBool("common.log_enable")
	utils.LogDir = viper.GetString("common.log_dir")
	utils.Cloudlist = viper.GetStringSlice("cloudlist")

	utils.BackdoorUser = fmt.Sprintf("%s %s %s",
		viper.GetString("backdoor-user.action"),
		viper.GetString("backdoor-user.username"),
		viper.GetString("backdoor-user.password"),
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

backdoor-user:
  action: add
  username: ctkguest
  password: 1QAZ2wsx@Asdlkj
`
