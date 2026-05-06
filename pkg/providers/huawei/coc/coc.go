// Package coc implements the cloudlist `vm` capability for Huawei Cloud via
// COC (Cloud Operations Center) BatchExecuteCommand. PLAN.md decision T3.2
// chose this path because COC most closely mirrors AWS SSM RunCommand /
// alibaba CloudAssistant semantics. UniAgent must be installed on the target
// ECS for executions to land.
//
// Endpoint paths are pattern-inferred from Huawei's documented v1 surface;
// verify against upstream COC docs before relying on this in production.
package coc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const (
	defaultRegion        = "cn-north-4"
	defaultExecuteUser   = "root"
	defaultExecuteScript = "SHELL"
	pollAttempts         = 30
	pollInterval         = 2 * time.Second
)

type Driver struct {
	Cred    auth.Credential
	Regions []string
	Client  *api.Client
	// nowSleep allows tests to short-circuit pollInterval without affecting
	// production timing.
	nowSleep func(time.Duration)
}

func (d *Driver) client() *api.Client {
	if d.Client == nil {
		d.Client = api.NewClient(d.Cred)
	}
	return d.Client
}

func (d *Driver) sleep(dur time.Duration) {
	if d.nowSleep != nil {
		d.nowSleep(dur)
		return
	}
	time.Sleep(dur)
}

func (d *Driver) region() string {
	for _, r := range d.Regions {
		if r = strings.TrimSpace(r); r != "" && r != "all" {
			return r
		}
	}
	if r := strings.TrimSpace(d.Cred.Region); r != "" && r != "all" {
		return r
	}
	return defaultRegion
}

// Execute submits the script to COC and polls until the order completes.
// Output is the per-instance stdout aggregated as a single string. An empty
// command short-circuits without making any API calls.
func (d *Driver) Execute(ctx context.Context, instanceID, command string) (schema.CommandResult, error) {
	if d == nil {
		return schema.CommandResult{}, errors.New("huawei coc: nil driver")
	}
	if strings.TrimSpace(command) == "" {
		return schema.CommandResult{}, errors.New("huawei coc: command is empty")
	}
	if strings.TrimSpace(instanceID) == "" {
		return schema.CommandResult{}, errors.New("huawei coc: instance id is empty")
	}
	region := d.region()
	body, err := json.Marshal(api.COCBatchExecuteCommandRequest{
		InstanceIDs:   []string{instanceID},
		Content:       wrapShell(command),
		ScriptType:    defaultExecuteScript,
		Username:      defaultExecuteUser,
		ExecutionMode: "INVOKE",
		TimeoutSec:    300,
	})
	if err != nil {
		return schema.CommandResult{}, err
	}
	submit, err := d.client().COCBatchExecuteCommand(ctx, region, body)
	if err != nil {
		return schema.CommandResult{}, fmt.Errorf("submit command: %w", err)
	}
	orderID := strings.TrimSpace(submit.OrderID)
	if orderID == "" {
		return schema.CommandResult{}, errors.New("huawei coc: empty order_id from submission")
	}
	for attempt := 0; attempt < pollAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return schema.CommandResult{}, ctx.Err()
		default:
		}
		status, err := d.client().COCDescribeJob(ctx, region, orderID)
		if err != nil {
			return schema.CommandResult{}, fmt.Errorf("poll order %s: %w", orderID, err)
		}
		if isTerminal(status.Status) {
			return schema.CommandResult{Output: aggregateOutput(status, orderID)}, nil
		}
		d.sleep(pollInterval)
	}
	return schema.CommandResult{Output: fmt.Sprintf("order %s still running after %d polls", orderID, pollAttempts)}, nil
}

// wrapShell ensures plain commands run under /bin/bash without requiring the
// caller to provide a shebang.
func wrapShell(command string) string {
	command = strings.TrimSpace(command)
	if strings.HasPrefix(command, "#!") {
		return command
	}
	return "#!/bin/bash\nset -e\n" + command + "\n"
}

func isTerminal(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "FINISHED", "SUCCESS", "FAILED", "CANCELED", "TIMEOUT":
		return true
	}
	return false
}

func aggregateOutput(resp api.COCDescribeJobResponse, orderID string) string {
	if len(resp.Results) == 0 {
		return fmt.Sprintf("order %s status=%s (no instance results)", orderID, resp.Status)
	}
	parts := make([]string, 0, len(resp.Results))
	for _, r := range resp.Results {
		header := fmt.Sprintf("[%s] %s", r.InstanceID, strings.TrimSpace(r.Status))
		body := strings.TrimSpace(r.Output)
		if body == "" {
			body = strings.TrimSpace(r.Message)
		}
		if body != "" {
			parts = append(parts, header+"\n"+body)
		} else {
			parts = append(parts, header)
		}
	}
	return strings.Join(parts, "\n---\n")
}
