package tat

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tat "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat/v20201028"
)

type Driver struct {
	Credential *common.Credential
	Region     string
}

var CacheHostList []schema.Host

func (d *Driver) RunCommand(instanceId, ostype, cmd string) string {
	cpf := profile.NewClientProfile()
	client, _ := tat.NewClient(d.Credential, d.Region, cpf)
	request := tat.NewRunCommandRequest()
	switch ostype {
	case "LINUX_UNIX":
		request.CommandType = common.StringPtr("SHELL")
	case "WINDOWS":
		request.CommandType = common.StringPtr("POWERSHELL")
	default:
		logger.Error("Unknown ostype", ostype)
		return ""
	}
	if strings.HasPrefix(cmd, "base64 ") {
		request.Content = common.StringPtr(strings.Split(cmd, " ")[1])
	} else {
		request.Content = common.StringPtr(base64.StdEncoding.EncodeToString([]byte(cmd)))
	}
	request.InstanceIds = common.StringPtrs([]string{instanceId})
	response, err := client.RunCommand(request)
	if err != nil {
		logger.Error(err)
		return ""
	}
	invid := *response.Response.InvocationId
	return describeInvocations(client, invid)
}

func describeInvocations(client *tat.Client, invid string) string {
	request := tat.NewDescribeInvocationsRequest()
	request.InvocationIds = common.StringPtrs([]string{invid})
	response, err := client.DescribeInvocations(request)
	if err != nil {
		logger.Error(err)
		return ""
	}
	taskId := *response.Response.InvocationSet[0].InvocationTaskBasicInfoSet[0].InvocationTaskId
	return describeInvocationTasks(client, taskId)
}

func describeInvocationTasks(client *tat.Client, taskId string) string {
	t := 0
	for {
		time.Sleep(1 * time.Second)
		t += 1
		request := tat.NewDescribeInvocationTasksRequest()
		request.InvocationTaskIds = common.StringPtrs([]string{taskId})
		request.HideOutput = common.BoolPtr(false)
		response, err := client.DescribeInvocationTasks(request)
		if err != nil {
			logger.Error(err)
			return ""
		}
		status := *response.Response.InvocationTaskSet[0].TaskStatus
		switch status {
		case "RUNNING", "PENDING", "DELIVERING":
			if t < 5 {
				continue
			}
			logger.Error("Timeout: Wait 5s by default.")
			return ""
		case "SUCCESS":
			output := *response.Response.InvocationTaskSet[0].TaskResult.Output
			raw, err := base64.StdEncoding.DecodeString(output)
			if err != nil {
				logger.Error(output, err)
				return ""
			}
			return string(raw)
		default:
			logger.Error("Exception status: " + status)
			return ""
		}
	}
}
