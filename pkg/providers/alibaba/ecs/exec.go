package ecs

import (
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

var CacheHostList []schema.Host

func RunCommand(client *ecs.Client, instanceId, region, ostype, cmd string) string {
	request := ecs.CreateRunCommandRequest()
	request.Scheme = "https"
	request.RegionId = region
	switch ostype {
	case "linux":
		request.Type = "RunShellScript"
	case "windows":
		request.Type = "RunBatScript"
	default:
		logger.Error("Unknown ostype", ostype)
		return ""
	}
	request.InstanceId = &[]string{instanceId}
	request.CommandContent = cmd
	response, err := client.RunCommand(request)
	if err != nil {
		logger.Error(err)
		return ""
	}
	cid := response.CommandId
	return describeInvocationResults(client, region, cid)
}

func describeInvocationResults(client *ecs.Client, region, cid string) string {
	t := 0
	for {
		time.Sleep(1 * time.Second)
		t += 1
		request := ecs.CreateDescribeInvocationResultsRequest()
		request.Scheme = "https"
		request.RegionId = region
		request.CommandId = cid
		request.ContentEncoding = "PlainText"
		response, err := client.DescribeInvocationResults(request)
		if err != nil {
			logger.Error(err)
			return ""
		}
		status := response.Invocation.InvocationResults.InvocationResult[0].InvokeRecordStatus
		switch status {
		case "Running":
			if t < 5 {
				continue
			}
			logger.Error("Timeout: Wait 5s by default.")
			return ""
		case "Finished":
			return response.Invocation.InvocationResults.InvocationResult[0].Output
		default:
			logger.Error("Exception status: " + status)
			return ""
		}
	}
}
