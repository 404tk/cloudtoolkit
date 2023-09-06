package ecs

import (
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
	request := ecs.CreateDescribeInvocationResultsRequest()
	request.Scheme = "https"
	request.RegionId = region
	request.CommandId = cid
	response, err := client.DescribeInvocationResults(request)
	if err != nil {
		return err.Error()
	}
	return response.Invocation.InvocationResults.InvocationResult[0].Output
}
