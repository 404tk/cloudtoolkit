// Package coc implements the cloudlist `vm` command capability for Huawei
// Cloud via COC (Cloud Operations Center) script jobs. UniAgent must be
// installed on the target ECS for executions to land.
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
	defaultRegion             = "cn-north-4"
	defaultLinuxExecuteUser   = "root"
	defaultWindowsExecuteUser = "Administrator"
	defaultLinuxExecuteScript = "SHELL"
	defaultWindowsScriptType  = "BAT"
	defaultTimeout            = 300
	defaultBatchIndex         = 1
	pollAttempts              = 30
	pollInterval              = 2 * time.Second
)

type Driver struct {
	Cred           auth.Credential
	Regions        []string
	DomainID       string
	Client         *api.Client
	ProjectCatalog *api.ProjectCatalog
	projectID      map[string]string
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
		if r != "" && r != "all" {
			return r
		}
	}
	if d.Cred.Region != "" && d.Cred.Region != "all" {
		return d.Cred.Region
	}
	return defaultRegion
}

// Execute creates a temporary COC script, executes it on one ECS instance, and
// polls until the execution reaches a terminal state. Output is aggregated from
// the per-instance execution log.
func (d *Driver) Execute(ctx context.Context, instanceID, command string) (schema.CommandResult, error) {
	return d.ExecuteOS(ctx, instanceID, "linux", command)
}

// ExecuteOS is Execute with an explicit target OS hint. Supported script types
// mirror Huawei COC: Linux uses SHELL and Windows uses BAT.
func (d *Driver) ExecuteOS(ctx context.Context, instanceID, osType, command string) (schema.CommandResult, error) {
	if d == nil {
		return schema.CommandResult{}, errors.New("huawei coc: nil driver")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(command) == "" {
		return schema.CommandResult{}, errors.New("huawei coc: command is empty")
	}
	if strings.TrimSpace(instanceID) == "" {
		return schema.CommandResult{}, errors.New("huawei coc: instance id is empty")
	}
	region := d.region()
	projectID, err := d.resolveProjectID(ctx, region)
	if err != nil {
		return schema.CommandResult{}, fmt.Errorf("resolve project id: %w", err)
	}
	scriptType, executeUser, content := scriptProfile(osType, command)
	createBody, err := json.Marshal(api.COCCreateScriptRequest{
		Name:        scriptName(),
		Properties:  api.COCScriptProperties{RiskLevel: "LOW", Version: "1.0"},
		Description: "CloudToolKit authorized validation script",
		Type:        scriptType,
		Content:     content,
	})
	if err != nil {
		return schema.CommandResult{}, err
	}
	create, err := d.client().COCCreateScript(ctx, region, projectID, createBody)
	if err != nil {
		return schema.CommandResult{}, fmt.Errorf("create script: %w", err)
	}
	scriptUUID := strings.TrimSpace(create.Data)
	if scriptUUID == "" {
		return schema.CommandResult{}, errors.New("huawei coc: empty script uuid from create script")
	}
	defer func() {
		_, _ = d.client().COCDeleteScript(context.WithoutCancel(ctx), region, projectID, scriptUUID)
	}()

	executeBody, err := json.Marshal(api.COCExecuteScriptRequest{
		ExecuteParam: api.COCScriptExecuteParam{
			Timeout:     defaultTimeout,
			SuccessRate: 100,
			ExecuteUser: executeUser,
		},
		ExecuteBatches: []api.COCExecuteInstancesBatchInfo{
			{
				BatchIndex: defaultBatchIndex,
				TargetInstances: []api.COCExecuteResourceInstance{
					{
						ResourceID: instanceID,
						RegionID:   region,
						Provider:   "ECS",
						Type:       "CLOUDSERVER",
					},
				},
				RotationStrategy: "CONTINUE",
			},
		},
	})
	if err != nil {
		return schema.CommandResult{}, err
	}
	execute, err := d.client().COCExecuteScript(ctx, region, projectID, scriptUUID, executeBody)
	if err != nil {
		return schema.CommandResult{}, fmt.Errorf("execute script %s: %w", scriptUUID, err)
	}
	executeUUID := strings.TrimSpace(execute.Data)
	if executeUUID == "" {
		return schema.CommandResult{}, errors.New("huawei coc: empty execute uuid from execute script")
	}
	for attempt := 0; attempt < pollAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return schema.CommandResult{}, ctx.Err()
		default:
		}
		job, err := d.client().COCGetScriptJobInfo(ctx, region, projectID, executeUUID)
		if err != nil {
			return schema.CommandResult{}, fmt.Errorf("poll execution %s: %w", executeUUID, err)
		}
		status := jobStatus(job)
		if isTerminal(status) {
			batchIndex := currentBatchIndex(job)
			batch, err := d.client().COCGetScriptJobBatch(ctx, region, projectID, executeUUID, batchIndex)
			if err != nil {
				return schema.CommandResult{}, fmt.Errorf("fetch execution %s batch %d: %w", executeUUID, batchIndex, err)
			}
			return schema.CommandResult{Output: aggregateBatchOutput(batch, executeUUID, status)}, nil
		}
		d.sleep(pollInterval)
	}
	return schema.CommandResult{Output: fmt.Sprintf("execution %s still running after %d polls", executeUUID, pollAttempts)}, nil
}

func scriptName() string {
	return fmt.Sprintf("ctk_validation_%d", time.Now().UTC().UnixNano())
}

func scriptProfile(osType, command string) (scriptType, executeUser, content string) {
	switch strings.ToLower(strings.TrimSpace(osType)) {
	case "windows":
		return defaultWindowsScriptType, defaultWindowsExecuteUser, wrapBatch(command)
	default:
		return defaultLinuxExecuteScript, defaultLinuxExecuteUser, wrapShell(command)
	}
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

func wrapBatch(command string) string {
	command = strings.TrimSpace(command)
	if strings.HasPrefix(strings.ToLower(command), "@echo off") {
		return command
	}
	return "@echo off\r\n" + command + "\r\n"
}

func (d *Driver) resolveProjectID(ctx context.Context, region string) (string, error) {
	if projectID, ok := d.ProjectCatalog.ProjectID(region); ok {
		return projectID, nil
	}
	if d.ProjectCatalog != nil {
		return "", &api.ProjectNotFoundError{Region: region}
	}
	if d.projectID == nil {
		d.projectID = make(map[string]string)
	}
	if cached := strings.TrimSpace(d.projectID[region]); cached != "" {
		return cached, nil
	}
	projectID, err := api.ResolveProjectID(ctx, d.client(), d.DomainID, region)
	if err != nil {
		return "", err
	}
	d.projectID[region] = projectID
	return projectID, nil
}

func isTerminal(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "FINISHED", "ABNORMAL", "CANCELED", "PAUSED", "SUCCESS", "FAILED", "TIMEOUT":
		return true
	}
	return false
}

func jobStatus(resp api.COCGetScriptJobInfoResponse) string {
	if resp.Data == nil {
		return ""
	}
	return strings.TrimSpace(resp.Data.Status)
}

func currentBatchIndex(resp api.COCGetScriptJobInfoResponse) int32 {
	if resp.Data != nil && resp.Data.Properties != nil && resp.Data.Properties.CurrentExecuteBatchIndex > 0 {
		return resp.Data.Properties.CurrentExecuteBatchIndex
	}
	return defaultBatchIndex
}

func aggregateBatchOutput(resp api.COCGetScriptJobBatchResponse, executeUUID, status string) string {
	if resp.Data == nil || len(resp.Data.ExecuteInstances) == 0 {
		return fmt.Sprintf("execution %s status=%s (no instance results)", executeUUID, status)
	}
	parts := make([]string, 0, len(resp.Data.ExecuteInstances))
	for _, r := range resp.Data.ExecuteInstances {
		instanceID := "unknown"
		if r.TargetInstance != nil && strings.TrimSpace(r.TargetInstance.ResourceID) != "" {
			instanceID = strings.TrimSpace(r.TargetInstance.ResourceID)
		}
		header := fmt.Sprintf("[%s] %s", instanceID, strings.TrimSpace(r.Status))
		body := strings.TrimSpace(r.Message)
		if body != "" {
			parts = append(parts, header+"\n"+body)
		} else {
			parts = append(parts, header)
		}
	}
	return strings.Join(parts, "\n---\n")
}
