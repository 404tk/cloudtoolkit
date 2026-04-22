package ecs

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) RunCommand(instanceID, osType, cmd string) string {
	ctx := context.Background()
	client, err := d.requireClient()
	if err != nil {
		logger.Error(err)
		return ""
	}

	commandType, ok := resolveCommandType(osType)
	if !ok {
		logger.Error("Unknown ostype", osType)
		return ""
	}

	region := d.requestRegion()
	if err := d.ensureCloudAssistantRunning(ctx, client, instanceID); err != nil {
		logger.Error(err)
		return ""
	}
	commandContent := base64.StdEncoding.EncodeToString([]byte(cmd))

	commandName := buildCloudAssistantName("ctk")
	createResp, err := client.CreateCommand(
		ctx,
		region,
		commandName,
		commandType,
		commandContent,
		"Base64",
	)
	if err != nil {
		logger.Error(err)
		return ""
	}

	commandID := strings.TrimSpace(createResp.Result.CommandID)
	if commandID == "" {
		logger.Error("Missing command id.")
		return ""
	}
	defer d.deleteCommand(ctx, client, commandID)

	invocationName := buildCloudAssistantName("ctk")
	invokeResp, err := client.InvokeCommand(
		ctx,
		region,
		commandID,
		invocationName,
		[]string{instanceID},
	)
	if err != nil {
		logger.Error(err)
		return ""
	}

	invocationID := strings.TrimSpace(invokeResp.Result.InvocationID)
	if invocationID == "" {
		logger.Error("Missing invocation id.")
		return ""
	}

	return d.describeInvocationResults(ctx, client, instanceID, commandID, invocationID)
}

func (d *Driver) ensureCloudAssistantRunning(ctx context.Context, client *api.Client, instanceID string) error {
	resp, err := client.DescribeCloudAssistantStatus(ctx, d.requestRegion(), []string{instanceID}, "", 20)
	if err != nil {
		return err
	}
	if len(resp.Result.Instances) == 0 {
		return fmt.Errorf("cloud assistant status unavailable for instance %s", instanceID)
	}

	instance := resp.Result.Instances[0]
	status := strings.ToUpper(strings.TrimSpace(instance.Status))
	if status == "RUNNING" {
		return nil
	}

	details := make([]string, 0, 3)
	if version := strings.TrimSpace(instance.ClientVersion); version != "" {
		details = append(details, "client-version="+version)
	}
	if heartbeat := strings.TrimSpace(instance.LastHeartbeatTime); heartbeat != "" {
		details = append(details, "last-heartbeat="+heartbeat)
	}
	if len(details) > 0 {
		return fmt.Errorf("cloud assistant agent status is %s, command execution requires RUNNING (%s)", status, strings.Join(details, ", "))
	}
	return fmt.Errorf("cloud assistant agent status is %s, command execution requires RUNNING", status)
}

func (d *Driver) describeInvocationResults(ctx context.Context, client *api.Client, instanceID, commandID, invocationID string) string {
	attempts := 0
	lastStatus := ""
	lastMessage := ""
	for {
		d.sleepFor(d.pollDelay())
		attempts++
		resp, err := client.DescribeInvocationResults(ctx, d.requestRegion(), invocationID, commandID, instanceID, 1)
		if err != nil {
			logger.Error(err)
			return ""
		}
		if len(resp.Result.InvocationResults) == 0 {
			if attempts < d.pollLimit() {
				continue
			}
			logger.Error("Missing invocation result.")
			return ""
		}

		result := resp.Result.InvocationResults[0]
		lastStatus = strings.ToUpper(result.Status())
		lastMessage = result.Message()
		switch status := strings.ToUpper(result.Status()); status {
		case "PENDING", "RUNNING", "CREATED", "DELIVERING", "IN_PROGRESS":
			if attempts < d.pollLimit() {
				continue
			}
			logger.Error("Timeout: Wait 20s by default. Last status:", lastStatus, lastMessage)
			return ""
		case "SUCCESS", "FINISHED", "SUCCEEDED":
			return decodeInvocationOutput(result.Output)
		default:
			if message := result.Message(); message != "" && result.ErrorCode != "" {
				logger.Error("Exception status: " + status + " - " + result.ErrorCode + " - " + message)
				return ""
			}
			if message := result.Message(); message != "" {
				logger.Error("Exception status: " + status + " - " + message)
				return ""
			}
			logger.Error("Exception status: " + status)
			return ""
		}
	}
}

func (d *Driver) deleteCommand(ctx context.Context, client *api.Client, commandID string) {
	if _, err := client.DeleteCommand(ctx, d.requestRegion(), commandID); err != nil {
		logger.Warning("Delete temporary command failed:", err)
	}
}

func resolveCommandType(osType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(osType)) {
	case "linux":
		return "Shell", true
	case "windows":
		return "PowerShell", true
	default:
		return "", false
	}
}

func decodeInvocationOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(output)
	if err != nil {
		return output
	}
	decoded := string(raw)
	return decoded
}

func buildCloudAssistantName(prefix string) string {
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

func (d *Driver) sleepFor(delay time.Duration) {
	if d.sleep != nil {
		d.sleep(delay)
		return
	}
	time.Sleep(delay)
}
