package ecs

import (
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

var (
	CacheHostList []schema.Host
	hostCacheMu   sync.RWMutex
)

func SetCacheHostList(hosts []schema.Host) {
	hostCacheMu.Lock()
	defer hostCacheMu.Unlock()
	CacheHostList = hosts
}

func GetCacheHostList() []schema.Host {
	hostCacheMu.RLock()
	defer hostCacheMu.RUnlock()
	return CacheHostList
}

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
	if strings.HasPrefix(cmd, "base64 ") {
		request.CommandContent = strings.Split(cmd, " ")[1]
		request.ContentEncoding = "Base64"
	}
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
