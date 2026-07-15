package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

const (
	defaultLROPollDelay = time.Second
	defaultLROMaxPolls  = 60
)

type runCommandTarget struct {
	SubscriptionID string
	ResourceGroup  string
	VMName         string
}

type runCommandPollResult struct {
	Status string `json:"status"`
	Error  struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Value      []azapi.RunCommandInstanceView `json:"value"`
	Properties struct {
		Status string                 `json:"status"`
		Output azapi.RunCommandResult `json:"output"`
	} `json:"properties"`
}

// RunCommand invokes Microsoft.Compute virtualMachines/runCommand on the
// instance identified by an ARM VM ID, `<subscription>/<resourceGroup>/<vmName>`,
// or the legacy `<resourceGroup>/<vmName>` shorthand. Caller's `osType` is used
// to pick the commandId — `linux` → RunShellScript, `windows` → RunPowerShellScript.
func (d *Driver) RunCommand(ctx context.Context, instanceID, osType, command string) (string, error) {
	if d == nil || d.Client == nil {
		return "", errors.New("azure compute: nil api client")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	target, err := d.resolveRunCommandTarget(instanceID)
	if err != nil {
		return "", err
	}
	commandID := pickCommandID(osType)
	body, err := json.Marshal(azapi.RunCommandInput{
		CommandID: commandID,
		Script:    splitScriptLines(command),
	})
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("api-version", azapi.ComputeAPIVersion)
	var result azapi.RunCommandResult
	meta, err := d.Client.DoWithResponse(ctx, azapi.Request{
		Method: http.MethodPost,
		Path: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s/runCommand",
			url.PathEscape(target.SubscriptionID), url.PathEscape(target.ResourceGroup), url.PathEscape(target.VMName)),
		Query:      query,
		Body:       body,
		Idempotent: false,
	}, &result)
	if err != nil {
		return "", err
	}
	if meta.StatusCode == http.StatusAccepted {
		return d.waitRunCommandLRO(ctx, meta)
	}
	return formatRunCommandOutput(result), nil
}

func pickCommandID(osType string) string {
	osType = strings.ToLower(strings.TrimSpace(osType))
	if osType == "windows" {
		return "RunPowerShellScript"
	}
	return "RunShellScript"
}

func splitScriptLines(command string) []string {
	out := make([]string, 0)
	for _, line := range strings.Split(command, "\n") {
		line = strings.TrimRight(line, "\r")
		if line != "" {
			out = append(out, line)
		}
	}
	if len(out) == 0 {
		return []string{command}
	}
	return out
}

func formatRunCommandOutput(r azapi.RunCommandResult) string {
	parts := make([]string, 0, len(r.Value))
	for _, v := range r.Value {
		if strings.TrimSpace(v.Message) != "" {
			parts = append(parts, v.Message)
			continue
		}
		if strings.TrimSpace(v.DisplayStatus) != "" {
			parts = append(parts, v.DisplayStatus)
		}
	}
	return strings.Join(parts, "\n")
}

func (d *Driver) resolveRunCommandTarget(instanceID string) (runCommandTarget, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return runCommandTarget{}, errors.New("azure compute: instance id required")
	}
	if strings.HasPrefix(instanceID, "/") || strings.Contains(strings.ToLower(instanceID), "/providers/microsoft.compute/virtualmachines/") {
		res, err := azapi.ParseResourceID(instanceID)
		if err != nil {
			return runCommandTarget{}, err
		}
		if !strings.EqualFold(res.Provider, "Microsoft.Compute") || !strings.EqualFold(res.ResourceType, "virtualMachines") {
			return runCommandTarget{}, fmt.Errorf("azure compute: ARM id must target Microsoft.Compute/virtualMachines, got %s/%s", res.Provider, res.ResourceType)
		}
		return runCommandTarget{
			SubscriptionID: res.SubscriptionID,
			ResourceGroup:  res.ResourceGroup,
			VMName:         res.ResourceName,
		}, nil
	}

	parts := strings.Split(strings.Trim(instanceID, "/"), "/")
	switch len(parts) {
	case 2:
		subscription, err := d.defaultSubscription()
		if err != nil {
			return runCommandTarget{}, err
		}
		if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			break
		}
		return runCommandTarget{SubscriptionID: subscription, ResourceGroup: parts[0], VMName: parts[1]}, nil
	case 3:
		if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[2]) == "" {
			break
		}
		return runCommandTarget{SubscriptionID: parts[0], ResourceGroup: parts[1], VMName: parts[2]}, nil
	}
	return runCommandTarget{}, fmt.Errorf("azure compute: instance id must be an ARM VM id, `<subscription>/<resourceGroup>/<vmName>`, or `<resourceGroup>/<vmName>`, got %q", instanceID)
}

func (d *Driver) defaultSubscription() (string, error) {
	for _, sub := range d.SubscriptionIDs {
		sub = strings.TrimSpace(sub)
		if sub != "" {
			return sub, nil
		}
	}
	return "", errors.New("azure compute: no subscription configured")
}

func (d *Driver) waitRunCommandLRO(ctx context.Context, meta azapi.ResponseMetadata) (string, error) {
	asyncURL := strings.TrimSpace(firstNonEmpty(meta.Header.Get("Azure-AsyncOperation"), meta.Header.Get("Operation-Location")))
	locationURL := strings.TrimSpace(meta.Header.Get("Location"))
	if asyncURL == "" && locationURL == "" {
		return "", errors.New("azure compute: runCommand returned 202 without Azure-AsyncOperation or Location header")
	}
	if asyncURL != "" {
		output, err := d.pollRunCommandURL(ctx, asyncURL, meta.Header.Get("Retry-After"))
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(output) != "" {
			return output, nil
		}
	}
	if locationURL != "" {
		return d.pollRunCommandURL(ctx, locationURL, meta.Header.Get("Retry-After"))
	}
	return "", nil
}

func (d *Driver) pollRunCommandURL(ctx context.Context, pollURL, retryAfter string) (string, error) {
	maxPolls := d.maxLROPolls()
	for attempt := 0; attempt < maxPolls; attempt++ {
		if attempt > 0 {
			if err := sleepPoll(ctx, retryAfter, d.pollDelay()); err != nil {
				return "", err
			}
		}
		var result runCommandPollResult
		meta, err := d.Client.DoWithResponse(ctx, azapi.Request{
			Method:     http.MethodGet,
			Path:       pollURL,
			Idempotent: true,
		}, &result)
		if err != nil {
			return "", err
		}
		retryAfter = meta.Header.Get("Retry-After")
		if output := result.output(); strings.TrimSpace(output) != "" {
			return output, nil
		}
		switch result.normalizedStatus() {
		case "", "succeeded":
			return "", nil
		case "failed", "canceled", "cancelled":
			return "", result.err()
		}
	}
	return "", fmt.Errorf("azure compute: runCommand operation did not complete after %d polls", maxPolls)
}

func (d *Driver) pollDelay() time.Duration {
	if d != nil && d.LROPollDelay > 0 {
		return d.LROPollDelay
	}
	return defaultLROPollDelay
}

func (d *Driver) maxLROPolls() int {
	if d != nil && d.LROMaxPolls > 0 {
		return d.LROMaxPolls
	}
	return defaultLROMaxPolls
}

func sleepPoll(ctx context.Context, retryAfter string, fallback time.Duration) error {
	if delay, ok := httpclient.ParseRetryAfter(retryAfter, time.Now()); ok {
		return httpclient.SleepWithContext(ctx, delay)
	}
	return httpclient.SleepWithContext(ctx, fallback)
}

func (r runCommandPollResult) normalizedStatus() string {
	status := strings.TrimSpace(r.Status)
	if status == "" {
		status = strings.TrimSpace(r.Properties.Status)
	}
	return strings.ToLower(status)
}

func (r runCommandPollResult) output() string {
	if len(r.Value) > 0 {
		return formatRunCommandOutput(azapi.RunCommandResult{Value: r.Value})
	}
	return formatRunCommandOutput(r.Properties.Output)
}

func (r runCommandPollResult) err() error {
	code := strings.TrimSpace(r.Error.Code)
	message := strings.TrimSpace(r.Error.Message)
	if code == "" && message == "" {
		return errors.New("azure compute: runCommand operation failed")
	}
	return fmt.Errorf("azure compute: runCommand operation failed: %s %s", code, message)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
