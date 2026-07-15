package tat

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
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
	output, err := d.RunCommandContext(context.Background(), instanceID, osType, cmd)
	if err != nil {
		logger.Error(err)
	}
	return output
}

// RunCommandContext is the cancellable provider-facing command path.
func (d *Driver) RunCommandContext(ctx context.Context, instanceID, osType, cmd string) (string, error) {
	if d == nil {
		return "", errors.New("tencent tat: nil driver")
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
		return "", fmt.Errorf("tencent tat: unsupported os type %q", osType)
	}
	response, err := client.RunTATCommand(
		ctx,
		d.Region,
		commandType,
		encodeContent(cmd),
		[]string{instanceID},
	)
	if err != nil {
		return "", err
	}
	invocationID := derefString(response.Response.InvocationID)
	if invocationID == "" {
		return "", errors.New("tencent tat: missing invocation id")
	}
	return d.describeInvocations(ctx, client, invocationID)
}

func (d *Driver) describeInvocations(ctx context.Context, client *api.Client, invocationID string) (string, error) {
	response, err := client.DescribeTATInvocations(ctx, d.Region, []string{invocationID})
	if err != nil {
		return "", err
	}
	if len(response.Response.InvocationSet) == 0 || len(response.Response.InvocationSet[0].InvocationTaskBasicInfoSet) == 0 {
		return "", errors.New("tencent tat: missing invocation task metadata")
	}
	taskID := derefString(response.Response.InvocationSet[0].InvocationTaskBasicInfoSet[0].InvocationTaskID)
	if taskID == "" {
		return "", errors.New("tencent tat: missing invocation task id")
	}
	return d.describeInvocationTasks(ctx, client, taskID)
}

func (d *Driver) describeInvocationTasks(ctx context.Context, client *api.Client, taskID string) (string, error) {
	attempts := 0
	for {
		if err := d.sleepFor(ctx, d.pollDelay()); err != nil {
			return "", err
		}
		attempts++
		response, err := client.DescribeTATInvocationTasks(ctx, d.Region, []string{taskID}, false)
		if err != nil {
			return "", err
		}
		if len(response.Response.InvocationTaskSet) == 0 {
			return "", errors.New("tencent tat: missing invocation task detail")
		}
		task := response.Response.InvocationTaskSet[0]
		status := strings.ToUpper(derefString(task.TaskStatus))
		switch status {
		case "RUNNING", "PENDING", "DELIVERING", "DELIVER_DELAYED":
			if attempts < d.pollLimit() {
				continue
			}
			return "", fmt.Errorf("tencent tat: Timeout: command did not complete after %d polls", d.pollLimit())
		case "SUCCESS":
			output := ""
			if task.TaskResult != nil {
				output = derefString(task.TaskResult.Output)
			}
			raw, err := base64.StdEncoding.DecodeString(output)
			if err != nil {
				return "", fmt.Errorf("tencent tat: decode command output: %w", err)
			}
			return string(raw), nil
		default:
			if msg := derefString(task.ErrorInfo); msg != "" {
				return "", fmt.Errorf("tencent tat: Exception status: %s - %s", status, msg)
			}
			return "", fmt.Errorf("tencent tat: Exception status: %s", status)
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

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
