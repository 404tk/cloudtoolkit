package tat

import (
	"context"
	"encoding/base64"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Credential      auth.Credential
	Region          string
	clientOptions   []api.Option
	pollInterval    time.Duration
	maxPollAttempts int
	sleep           func(time.Duration)
}

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

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Credential, d.clientOptions...)
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) RunCommand(instanceID, osType, cmd string) string {
	ctx := context.Background()
	client := d.newClient()
	commandType, ok := resolveCommandType(osType)
	if !ok {
		logger.Error("Unknown ostype", osType)
		return ""
	}
	response, err := client.RunTATCommand(
		ctx,
		d.Region,
		commandType,
		encodeContent(cmd),
		[]string{instanceID},
	)
	if err != nil {
		logger.Error(err)
		return ""
	}
	invocationID := derefString(response.Response.InvocationID)
	if invocationID == "" {
		logger.Error("Missing invocation id.")
		return ""
	}
	return d.describeInvocations(ctx, client, invocationID)
}

func (d *Driver) describeInvocations(ctx context.Context, client *api.Client, invocationID string) string {
	response, err := client.DescribeTATInvocations(ctx, d.Region, []string{invocationID})
	if err != nil {
		logger.Error(err)
		return ""
	}
	if len(response.Response.InvocationSet) == 0 || len(response.Response.InvocationSet[0].InvocationTaskBasicInfoSet) == 0 {
		logger.Error("Missing invocation task metadata.")
		return ""
	}
	taskID := derefString(response.Response.InvocationSet[0].InvocationTaskBasicInfoSet[0].InvocationTaskID)
	if taskID == "" {
		logger.Error("Missing invocation task id.")
		return ""
	}
	return d.describeInvocationTasks(ctx, client, taskID)
}

func (d *Driver) describeInvocationTasks(ctx context.Context, client *api.Client, taskID string) string {
	attempts := 0
	for {
		d.sleepFor(d.pollDelay())
		attempts++
		response, err := client.DescribeTATInvocationTasks(ctx, d.Region, []string{taskID}, false)
		if err != nil {
			logger.Error(err)
			return ""
		}
		if len(response.Response.InvocationTaskSet) == 0 {
			logger.Error("Missing invocation task detail.")
			return ""
		}
		task := response.Response.InvocationTaskSet[0]
		status := strings.ToUpper(derefString(task.TaskStatus))
		switch status {
		case "RUNNING", "PENDING", "DELIVERING", "DELIVER_DELAYED":
			if attempts < d.pollLimit() {
				continue
			}
			logger.Error("Timeout: Wait 5s by default.")
			return ""
		case "SUCCESS":
			output := ""
			if task.TaskResult != nil {
				output = derefString(task.TaskResult.Output)
			}
			raw, err := base64.StdEncoding.DecodeString(output)
			if err != nil {
				logger.Error(output, err)
				return ""
			}
			return string(raw)
		default:
			if msg := derefString(task.ErrorInfo); msg != "" {
				logger.Error("Exception status: " + status + " - " + msg)
				return ""
			}
			logger.Error("Exception status: " + status)
			return ""
		}
	}
}

func resolveCommandType(osType string) (string, bool) {
	switch osType {
	case "LINUX_UNIX", "linux":
		return "SHELL", true
	case "WINDOWS", "windows":
		return "POWERSHELL", true
	default:
		return "", false
	}
}

func encodeContent(cmd string) string {
	if strings.HasPrefix(cmd, "base64 ") {
		return strings.TrimPrefix(cmd, "base64 ")
	}
	return base64.StdEncoding.EncodeToString([]byte(cmd))
}

func (d *Driver) pollDelay() time.Duration {
	if d.pollInterval > 0 {
		return d.pollInterval
	}
	return time.Second
}

func (d *Driver) pollLimit() int {
	if d.maxPollAttempts > 0 {
		return d.maxPollAttempts
	}
	return 20
}

func (d *Driver) sleepFor(delay time.Duration) {
	if d.sleep != nil {
		d.sleep(delay)
		return
	}
	time.Sleep(delay)
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
