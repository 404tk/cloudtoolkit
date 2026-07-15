package ecs

import (
	"context"
	"errors"
	"fmt"
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
	output, err := d.RunCommandContext(context.Background(), instanceID, osType, cmd)
	if err != nil {
		logger.Error(err)
	}
	return output
}

// RunCommandContext is the cancellable form used by provider capabilities.
// RunCommand remains as a compatibility wrapper for existing integrations.
func (d *Driver) RunCommandContext(ctx context.Context, instanceID, osType, cmd string) (string, error) {
	if d == nil {
		return "", errors.New("alibaba ecs: nil driver")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	client := d.newClient()
	commandType, ok := resolveCommandType(osType)
	if !ok {
		return "", fmt.Errorf("alibaba ecs: unsupported os type %q", osType)
	}

	contentEncoding := ""
	commandContent := cmd
	if strings.HasPrefix(cmd, "base64 ") {
		commandContent = strings.TrimSpace(strings.TrimPrefix(cmd, "base64 "))
		contentEncoding = "Base64"
	}

	response, err := client.RunECSCommand(ctx, d.Region, commandType, commandContent, contentEncoding, []string{instanceID})
	if err != nil {
		return "", err
	}
	if response.CommandID == "" {
		return "", errors.New("alibaba ecs: missing command id")
	}
	return d.describeInvocationResults(ctx, client, response.CommandID)
}

func (d *Driver) describeInvocationResults(ctx context.Context, client interface {
	DescribeECSInvocationResults(context.Context, string, string) (api.DescribeECSInvocationResultsResponse, error)
}, commandID string) (string, error) {
	attempts := 0
	for {
		if err := d.sleepFor(ctx, d.pollDelay()); err != nil {
			return "", err
		}
		attempts++

		response, err := client.DescribeECSInvocationResults(ctx, d.Region, commandID)
		if err != nil {
			return "", err
		}
		if len(response.Invocation.InvocationResults.InvocationResult) == 0 {
			return "", errors.New("alibaba ecs: missing invocation result")
		}

		result := response.Invocation.InvocationResults.InvocationResult[0]
		switch status := result.InvokeRecordStatus; status {
		case "Running":
			if attempts < d.pollLimit() {
				continue
			}
			return "", fmt.Errorf("alibaba ecs: Timeout: command did not complete after %d polls", d.pollLimit())
		case "Finished":
			return result.Output, nil
		default:
			if result.ErrorInfo != "" {
				return "", fmt.Errorf("alibaba ecs: Exception status: %s - %s", status, result.ErrorInfo)
			}
			return "", fmt.Errorf("alibaba ecs: Exception status: %s", status)
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
	return 20
}

func (d *Driver) sleepFor(ctx context.Context, delay time.Duration) error {
	if d.sleep != nil {
		d.sleep(delay)
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
