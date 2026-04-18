package ecs

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
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

func (d *Driver) RunCommand(instanceID, osType, cmd string) string {
	ctx := context.Background()
	client := d.newClient()
	commandType, ok := resolveCommandType(osType)
	if !ok {
		logger.Error("Unknown ostype", osType)
		return ""
	}

	contentEncoding := ""
	commandContent := cmd
	if strings.HasPrefix(cmd, "base64 ") {
		commandContent = strings.TrimSpace(strings.TrimPrefix(cmd, "base64 "))
		contentEncoding = "Base64"
	}

	response, err := client.RunECSCommand(ctx, d.Region, commandType, commandContent, contentEncoding, []string{instanceID})
	if err != nil {
		logger.Error(err)
		return ""
	}
	if response.CommandID == "" {
		logger.Error("Missing command id.")
		return ""
	}
	return d.describeInvocationResults(ctx, client, response.CommandID)
}

func (d *Driver) describeInvocationResults(ctx context.Context, client interface {
	DescribeECSInvocationResults(context.Context, string, string) (api.DescribeECSInvocationResultsResponse, error)
}, commandID string) string {
	attempts := 0
	for {
		d.sleepFor(d.pollDelay())
		attempts++

		response, err := client.DescribeECSInvocationResults(ctx, d.Region, commandID)
		if err != nil {
			logger.Error(err)
			return ""
		}
		if len(response.Invocation.InvocationResults.InvocationResult) == 0 {
			logger.Error("Missing invocation result.")
			return ""
		}

		result := response.Invocation.InvocationResults.InvocationResult[0]
		switch status := result.InvokeRecordStatus; status {
		case "Running":
			if attempts < d.pollLimit() {
				continue
			}
			logger.Error("Timeout: Wait 5s by default.")
			return ""
		case "Finished":
			return result.Output
		default:
			if result.ErrorInfo != "" {
				logger.Error("Exception status: " + status + " - " + result.ErrorInfo)
				return ""
			}
			logger.Error("Exception status: " + status)
			return ""
		}
	}
}

func resolveCommandType(osType string) (string, bool) {
	switch osType {
	case "linux":
		return "RunShellScript", true
	case "windows":
		return "RunBatScript", true
	default:
		return "", false
	}
}

func (d *Driver) pollDelay() time.Duration {
	if d.pollInterval > 0 {
		return d.pollInterval
	}
	return time.Second
}

func (d *Driver) pollLimit() int {
	if d.maxPolls > 0 {
		return d.maxPolls
	}
	return 5
}

func (d *Driver) sleepFor(delay time.Duration) {
	if d.sleep != nil {
		d.sleep(delay)
		return
	}
	time.Sleep(delay)
}
