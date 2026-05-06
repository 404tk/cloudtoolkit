// Package vmexec implements the cloudlist `vm` capability for GCP via the
// metadata startup-script + reboot path (PLAN.md decision T2.2/Task 10).
//
// GCP has no managed exec service equivalent to AWS SSM RunCommand or
// alibaba CloudAssistant. The portable management-plane approach is:
//
//  1. instance.get to read current metadata + fingerprint
//  2. instance.setMetadata to add a `startup-script` key with the command
//  3. instance.reset to trigger the startup script on next boot
//
// We do NOT capture stdout — startup-script output goes to the serial
// console which requires a separate IAM permission to read. The instance-cmd
// payload surfaces "command queued" with a clear note. Operators should
// follow up with `gcloud compute instances get-serial-port-output` to read
// the result. This is documented in the demo replay.
package vmexec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const startupScriptKey = "startup-script"

type Driver struct {
	Projects []string
	Client   *api.Client
}

// Execute writes a startup-script and reboots `instanceID`. instanceID may be
// `<zone>/<instance>` (preferred) or just `<instance>` (driver scans the first
// configured project's zones for a matching instance).
func (d *Driver) Execute(ctx context.Context, instanceID, command string) (schema.CommandResult, error) {
	if d == nil || d.Client == nil {
		return schema.CommandResult{}, errors.New("gcp compute: nil api client")
	}
	if strings.TrimSpace(command) == "" {
		return schema.CommandResult{}, errors.New("gcp compute: command is empty")
	}
	project, zone, instance, err := d.resolveTarget(ctx, instanceID)
	if err != nil {
		return schema.CommandResult{}, err
	}
	current, err := d.getInstance(ctx, project, zone, instance)
	if err != nil {
		return schema.CommandResult{}, fmt.Errorf("get instance: %w", err)
	}
	updated := mergeStartupScript(current.Metadata, wrapStartupScript(command))
	if err := d.setMetadata(ctx, project, zone, instance, updated); err != nil {
		return schema.CommandResult{}, fmt.Errorf("setMetadata: %w", err)
	}
	if err := d.reset(ctx, project, zone, instance); err != nil {
		return schema.CommandResult{}, fmt.Errorf("reset: %w", err)
	}
	return schema.CommandResult{
		Output: fmt.Sprintf(
			"Command queued via startup-script on projects/%s/zones/%s/instances/%s. "+
				"Reboot triggered; read serial console for output.",
			project, zone, instance,
		),
	}, nil
}

func (d *Driver) resolveTarget(ctx context.Context, raw string) (string, string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", errors.New("gcp compute: instance id is empty")
	}
	// Accept the explicit "<zone>/<instance>" form first — it's deterministic.
	if zone, instance, ok := strings.Cut(raw, "/"); ok {
		return d.firstProject(), strings.TrimSpace(zone), strings.TrimSpace(instance), nil
	}
	// Otherwise fall back to scanning zones in the first project for a name
	// match; this is convenient for the demo flow where the instance name is
	// globally unique within a tiny fixture set.
	project := d.firstProject()
	if project == "" {
		return "", "", "", errors.New("gcp compute: no project configured")
	}
	zones, err := d.listZones(ctx, project)
	if err != nil {
		return "", "", "", err
	}
	for _, z := range zones {
		instances, err := d.listInstanceNames(ctx, project, z.Name)
		if err != nil {
			continue
		}
		for _, name := range instances {
			if name == raw {
				return project, z.Name, name, nil
			}
		}
	}
	return "", "", "", fmt.Errorf("gcp compute: instance %q not found in project %s", raw, project)
}

func (d *Driver) firstProject() string {
	for _, p := range d.Projects {
		if p = strings.TrimSpace(p); p != "" {
			return p
		}
	}
	return ""
}

func (d *Driver) getInstance(ctx context.Context, project, zone, instance string) (api.InstanceWithMetadata, error) {
	var resp api.InstanceWithMetadata
	err := d.Client.Do(ctx, api.Request{
		Method:     http.MethodGet,
		BaseURL:    api.ComputeBaseURL,
		Path:       fmt.Sprintf("/compute/v1/projects/%s/zones/%s/instances/%s", url.PathEscape(project), url.PathEscape(zone), url.PathEscape(instance)),
		Idempotent: true,
	}, &resp)
	return resp, err
}

func (d *Driver) setMetadata(ctx context.Context, project, zone, instance string, metadata api.InstanceMetadata) error {
	body, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	var op api.ComputeOperation
	if err := d.Client.Do(ctx, api.Request{
		Method:  http.MethodPost,
		BaseURL: api.ComputeBaseURL,
		Path:    fmt.Sprintf("/compute/v1/projects/%s/zones/%s/instances/%s/setMetadata", url.PathEscape(project), url.PathEscape(zone), url.PathEscape(instance)),
		Body:    body,
	}, &op); err != nil {
		return err
	}
	return operationError(op)
}

func (d *Driver) reset(ctx context.Context, project, zone, instance string) error {
	var op api.ComputeOperation
	if err := d.Client.Do(ctx, api.Request{
		Method:  http.MethodPost,
		BaseURL: api.ComputeBaseURL,
		Path:    fmt.Sprintf("/compute/v1/projects/%s/zones/%s/instances/%s/reset", url.PathEscape(project), url.PathEscape(zone), url.PathEscape(instance)),
		// Empty body — reset takes no parameters.
	}, &op); err != nil {
		return err
	}
	return operationError(op)
}

func operationError(op api.ComputeOperation) error {
	if op.Error == nil || len(op.Error.Errors) == 0 {
		return nil
	}
	parts := make([]string, 0, len(op.Error.Errors))
	for _, e := range op.Error.Errors {
		parts = append(parts, fmt.Sprintf("%s: %s", e.Code, e.Message))
	}
	return errors.New("compute operation failed: " + strings.Join(parts, "; "))
}

// mergeStartupScript replaces the existing startup-script (if any) without
// touching other metadata keys, preserving the fingerprint for the
// optimistic-concurrency setMetadata call.
func mergeStartupScript(existing api.InstanceMetadata, script string) api.InstanceMetadata {
	out := api.InstanceMetadata{
		Fingerprint: existing.Fingerprint,
		Items:       make([]api.InstanceMetadataItem, 0, len(existing.Items)+1),
	}
	for _, item := range existing.Items {
		if item.Key == startupScriptKey {
			continue
		}
		out.Items = append(out.Items, item)
	}
	out.Items = append(out.Items, api.InstanceMetadataItem{Key: startupScriptKey, Value: script})
	return out
}

// wrapStartupScript adds a small shebang so plain `cmd` strings work even when
// callers pass single-line shell snippets without explicit interpreter.
func wrapStartupScript(command string) string {
	command = strings.TrimSpace(command)
	if strings.HasPrefix(command, "#!") {
		return command
	}
	return "#!/bin/bash\nset -e\n" + command + "\n"
}

// listZones / listInstanceNames are minimal helpers used by the resolver path.
func (d *Driver) listZones(ctx context.Context, project string) ([]api.Zone, error) {
	var resp api.ListZonesResponse
	err := d.Client.Do(ctx, api.Request{
		Method:     http.MethodGet,
		BaseURL:    api.ComputeBaseURL,
		Path:       fmt.Sprintf("/compute/v1/projects/%s/zones", url.PathEscape(project)),
		Idempotent: true,
	}, &resp)
	return resp.Items, err
}

func (d *Driver) listInstanceNames(ctx context.Context, project, zone string) ([]string, error) {
	var resp api.ListInstancesResponse
	err := d.Client.Do(ctx, api.Request{
		Method:     http.MethodGet,
		BaseURL:    api.ComputeBaseURL,
		Path:       fmt.Sprintf("/compute/v1/projects/%s/zones/%s/instances", url.PathEscape(project), url.PathEscape(zone)),
		Idempotent: true,
	}, &resp)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(resp.Items))
	for _, i := range resp.Items {
		out = append(out, i.Name)
	}
	return out, nil
}
