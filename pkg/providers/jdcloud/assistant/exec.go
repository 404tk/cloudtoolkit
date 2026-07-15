package assistant

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

// Driver wraps JDCloud's Cloud Assistant (云助手) command-exec flow:
// CreateCommand → InvokeCommand → poll DescribeInvocations → DeleteCommands.
// JDCloud does not expose an agent-status preflight (unlike volcengine's
// DescribeCloudAssistantStatus), so an offline agent surfaces as an invalid /
// failed invocation status and we relay the ErrorInfo upstream.
type Driver struct {
	Client       *api.Client
	Region       string
	pollInterval time.Duration
	maxPolls     int
	sleep        func(time.Duration)
}

var errNilAPIClient = errors.New("jdcloud assistant: nil api client")

func (d *Driver) RunCommand(instanceID, osType, cmd string) string {
	output, err := d.RunCommandContext(context.Background(), instanceID, osType, cmd)
	if err != nil {
		logger.Error(err)
	}
	return output
}

// RunCommandContext is the cancellable Cloud Assistant execution path used by
// the provider capability. RunCommand is kept for compatibility.
func (d *Driver) RunCommandContext(ctx context.Context, instanceID, osType, cmd string) (string, error) {
	if d == nil {
		return "", errNilAPIClient
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if d.Client == nil {
		return "", errNilAPIClient
	}

	region := strings.TrimSpace(d.Region)
	if region == "" {
		return "", errors.New("jdcloud assistant: empty region; run `cloudlist` to populate the host cache or set a region explicitly")
	}

	commandType := resolveCommandType(osType)

	commandContent := base64.StdEncoding.EncodeToString([]byte(cmd))
	commandName := buildCommandName("ctk")

	commandID, err := d.createCommand(ctx, region, commandName, commandType, commandContent)
	if err != nil {
		return "", err
	}
	if commandID == "" {
		return "", errors.New("jdcloud assistant: missing command id")
	}
	// The temporary command is always cleaned up; the console shell dispatches
	// a fresh CreateCommand per keystroke so leaving these around would leak
	// into the customer's command library and hit per-account quotas.
	defer d.deleteCommand(context.WithoutCancel(ctx), region, commandID)

	invokeID, err := d.invokeCommand(ctx, region, commandID, instanceID)
	if err != nil {
		return "", err
	}
	if invokeID == "" {
		return "", errors.New("jdcloud assistant: missing invocation id")
	}

	return d.pollInvocation(ctx, region, invokeID)
}

func (d *Driver) createCommand(ctx context.Context, region, name, commandType, contentB64 string) (string, error) {
	body, err := json.Marshal(api.CreateCommandRequest{
		RegionID:       region,
		CommandName:    name,
		CommandType:    commandType,
		CommandContent: contentB64,
	})
	if err != nil {
		return "", err
	}
	var resp api.CreateCommandResponse
	if err := d.Client.DoJSON(ctx, api.Request{
		Service: "assistant",
		Region:  region,
		Method:  http.MethodPost,
		Version: "v1",
		Path:    "/regions/" + region + "/createCommand",
		Body:    body,
	}, &resp); err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Result.CommandID), nil
}

func (d *Driver) invokeCommand(ctx context.Context, region, commandID, instanceID string) (string, error) {
	body, err := json.Marshal(api.InvokeCommandRequest{
		RegionID:  region,
		CommandID: commandID,
		Instances: []string{instanceID},
	})
	if err != nil {
		return "", err
	}
	var resp api.InvokeCommandResponse
	if err := d.Client.DoJSON(ctx, api.Request{
		Service: "assistant",
		Region:  region,
		Method:  http.MethodPost,
		Version: "v1",
		Path:    "/regions/" + region + "/invokeCommand",
		Body:    body,
	}, &resp); err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Result.InvokeID), nil
}

func (d *Driver) pollInvocation(ctx context.Context, region, invokeID string) (string, error) {
	attempts := 0
	for {
		if err := d.sleepFor(ctx, d.pollDelay()); err != nil {
			return "", err
		}
		attempts++
		body, err := json.Marshal(api.DescribeInvocationsRequest{
			RegionID:   region,
			PageNumber: 1,
			PageSize:   1,
			InvokeIDs:  []string{invokeID},
		})
		if err != nil {
			return "", err
		}
		var resp api.DescribeInvocationsResponse
		if err := d.Client.DoJSON(ctx, api.Request{
			Service: "assistant",
			Region:  region,
			Method:  http.MethodPost,
			Version: "v1",
			Path:    "/regions/" + region + "/describeInvocations",
			Body:    body,
		}, &resp); err != nil {
			return "", err
		}
		if len(resp.Result.Invocations) == 0 {
			if attempts < d.pollLimit() {
				continue
			}
			return "", fmt.Errorf("jdcloud assistant: missing invocation record after %d polls", d.pollLimit())
		}

		inv := resp.Result.Invocations[0]
		status := strings.ToLower(strings.TrimSpace(inv.Status))
		switch status {
		case "waiting", "pending", "running", "stopping":
			if attempts < d.pollLimit() {
				continue
			}
			return "", fmt.Errorf("jdcloud assistant: command did not complete after %d polls (last status %s)", d.pollLimit(), status)
		case "finish":
			return decodeOutput(invocationOutput(inv)), nil
		default:
			// failed / partial_failed / stopped / per-instance invalid / timeout
			// / terminated / aborted / cancel / error all land here. Surface the
			// ErrorInfo — that's where "agent offline" / "instance unreachable"
			// signals live since JDCloud has no agent-status preflight.
			if info := invocationErrorInfo(inv); info != "" {
				return "", fmt.Errorf("jdcloud assistant: command status %s: %s", status, info)
			}
			return "", fmt.Errorf("jdcloud assistant: command status %s", status)
		}
	}
}

func (d *Driver) deleteCommand(ctx context.Context, region, commandID string) {
	body, err := json.Marshal(api.DeleteCommandsRequest{
		RegionID:   region,
		CommandIDs: []string{commandID},
	})
	if err != nil {
		logger.Warning("Delete temporary command failed:", err)
		return
	}
	// Cleanup runs on a fresh short context so it can still proceed when the
	// main ctx has been cancelled (timeout or SIGINT).
	cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var resp api.DeleteCommandsResponse
	if err := d.Client.DoJSON(cleanupCtx, api.Request{
		Service: "assistant",
		Region:  region,
		Method:  http.MethodPost,
		Version: "v1",
		Path:    "/regions/" + region + "/deleteCommands",
		Body:    body,
	}, &resp); err != nil {
		logger.Warning("Delete temporary command failed:", err)
	}
}

func invocationOutput(inv api.Invocation) string {
	if len(inv.InvocationInstances) == 0 {
		return ""
	}
	return inv.InvocationInstances[0].Output
}

func invocationErrorInfo(inv api.Invocation) string {
	if len(inv.InvocationInstances) > 0 {
		if info := strings.TrimSpace(inv.InvocationInstances[0].ErrorInfo); info != "" {
			return info
		}
	}
	if info := strings.TrimSpace(inv.ErrorInfo); info != "" {
		return info
	}
	if len(inv.InvocationInstances) > 0 {
		if status := strings.TrimSpace(inv.InvocationInstances[0].Status); status != "" {
			return "instance_status=" + status
		}
	}
	return ""
}

func resolveCommandType(osType string) string {
	switch strings.ToLower(strings.TrimSpace(osType)) {
	case "linux":
		return "shell"
	case "windows":
		return "powershell"
	default:
		return "shell"
	}
}

// decodeOutput trims and base64-decodes the invocation output. JDCloud truncates
// bodies > 6000B to first-5000B + last-1000B concatenated, so a decode failure
// falls back to the raw string rather than surfacing an empty value.
func decodeOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(output)
	if err != nil {
		return output
	}
	return string(raw)
}

func buildCommandName(prefix string) string {
	upper := big.NewInt(100000)
	n, err := rand.Int(rand.Reader, upper)
	if err != nil {
		return fmt.Sprintf("%s-%05d", prefix, time.Now().UTC().UnixNano()%100000)
	}
	return fmt.Sprintf("%s-%05d", prefix, n.Int64())
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
