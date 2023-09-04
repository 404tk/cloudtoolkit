package ram

import (
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

func (d *Driver) DelUser() {
	client := d.NewClient()
	err := detachPolicyFromUser(client, d.UserName)
	if err != nil {
		if !strings.Contains(err.Error(), "EntityNotExist") {
			logger.Error(fmt.Sprintf("Remove policy from %s failed: %s\n", d.UserName, err))
			return
		}
	}
	err = deleteUser(client, d.UserName)
	if err != nil {
		logger.Error(fmt.Sprintf("Delete user %s failed: %s\n", d.UserName, err))
		return
	}
	logger.Warning("Done.")
}

func detachPolicyFromUser(client *ram.Client, userName string) error {
	request := ram.CreateDetachPolicyFromUserRequest()
	request.Scheme = "https"
	request.PolicyType = "System"
	request.PolicyName = "AdministratorAccess"
	request.UserName = userName
	_, err := client.DetachPolicyFromUser(request)
	return err
}

func deleteUser(client *ram.Client, userName string) error {
	request := ram.CreateDeleteUserRequest()
	request.Scheme = "https"
	request.UserName = userName
	_, err := client.DeleteUser(request)
	return err
}
