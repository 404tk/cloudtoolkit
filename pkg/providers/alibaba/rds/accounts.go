package rds

import (
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
)

func (d *Driver) CreateAccount(instanceId, dbname string) bool {
	client, err := d.NewClient()
	if err != nil {
		logger.Error(err)
		return false
	}
	account := strings.Split(utils.DBAccount, ":")
	request := rds.CreateCreateAccountRequest()
	request.Scheme = "https"
	request.DBInstanceId = instanceId
	request.AccountName = account[0]
	request.AccountPassword = account[1]
	request.AccountType = "Normal"
	_, err = client.CreateAccount(request)
	if err != nil {
		logger.Error(err)
		return false
	}
	err = grantAccountPrivilege(client, instanceId, account[0], dbname)
	if err != nil {
		logger.Error(err)
		return false
	}
	fmt.Printf("\n%-10s\t%-10s\t%-60s\n", "Username", "Password", "Privilege")
	fmt.Printf("%-10s\t%-10s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-10s\t%-60s\n\n",
		account[0], account[1], "ReadOnly")
	return true
}

func (d *Driver) DeleteAccount(instanceId string) {
	client, err := d.NewClient()
	if err != nil {
		logger.Error(err)
		return
	}
	account := strings.Split(utils.DBAccount, ":")
	request := rds.CreateDeleteAccountRequest()
	request.Scheme = "https"
	request.DBInstanceId = instanceId
	request.AccountName = account[0]
	resp, err := client.DeleteAccount(request)
	if err != nil {
		logger.Error(err)
		return
	}
	if resp.IsSuccess() {
		logger.Warning(account[0] + " user delete completed.")
	}
}

func grantAccountPrivilege(client *rds.Client, instanceId, uname, dbname string) error {
	request := rds.CreateGrantAccountPrivilegeRequest()
	request.Scheme = "https"
	request.DBInstanceId = instanceId
	request.AccountName = uname
	request.DBName = dbname
	request.AccountPrivilege = "ReadOnly"
	_, err := client.GrantAccountPrivilege(request)
	return err
}

/*
func describeAccounts(instanceId string) {
	request := rds.CreateDescribeAccountsRequest()
	request.Scheme = "https"
	request.DBInstanceId = instanceId
	//response, err := client.DescribeAccounts(request)
	//AccountStatus
	//AccountDescription
	//AccountType
	//AccountName
	//DatabasePrivileges
}
*/
