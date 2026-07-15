package ssm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

// CacheHostList mirrors the alibaba/tencent/volcengine pattern: cloudlist
// populates a cache so the REPL `shell <id>` flow can resolve region + OS
// type without a second API roundtrip.
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

type Driver struct {
	Client *api.Client
	Region string

	// pollInterval / maxPolls / sleep are wired up by tests so the polling
	// loop does not actually wait. Production paths use the defaults.
	pollInterval time.Duration
	maxPolls     int
	sleep        func(time.Duration)
}

// SetPollOptions overrides the default polling cadence. Used by tests; the
// production driver should keep the defaults (1s interval, ~20 polls).
func (d *Driver) SetPollOptions(interval time.Duration, max int, sleep func(time.Duration)) {
	d.pollInterval = interval
	d.maxPolls = max
	d.sleep = sleep
}

// RunCommand sends `cmd` to instanceID via SSM and polls for completion.
// Returns the command stdout (or stderr if exit was non-zero). Empty string
// on hard failure — callers log via logger; the REPL surface is the same as
// the existing alibaba/tencent ECS exec drivers.
func (d *Driver) RunCommand(instanceID, osType, cmd string) string {
	output, err := d.RunCommandContext(context.Background(), instanceID, osType, cmd)
	if err != nil {
		logger.Error(err)
	}
	return output
}

// RunCommandContext sends a command and polls SSM until completion. The
// context is passed to every API request and also controls the poll delay, so
// caller deadlines and interrupts stop remote work promptly.
func (d *Driver) RunCommandContext(ctx context.Context, instanceID, osType, cmd string) (string, error) {
	if d == nil || d.Client == nil {
		return "", errors.New("aws ssm: nil client")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	doc, ok := resolveDocumentName(osType)
	if !ok {
		return "", fmt.Errorf("aws ssm: unsupported os type %q", osType)
	}
	region := strings.TrimSpace(d.Region)
	if region == "" || region == "all" {
		return "", errors.New("aws ssm: empty region")
	}
	resp, err := d.Client.SSMSendCommand(ctx, region, doc, []string{instanceID}, []string{cmd})
	if err != nil {
		return "", err
	}
	commandID := strings.TrimSpace(resp.Command.CommandID)
	if commandID == "" {
		return "", errors.New("aws ssm: empty command id")
	}
	return d.pollInvocation(ctx, region, commandID, instanceID)
}

func (d *Driver) pollInvocation(ctx context.Context, region, commandID, instanceID string) (string, error) {
	for attempts := 0; attempts < d.pollLimit(); attempts++ {
		if err := d.sleepFor(ctx, d.pollDelay()); err != nil {
			return "", err
		}
		invocation, err := d.Client.SSMGetCommandInvocation(ctx, region, commandID, instanceID)
		if err != nil {
			// SSM returns InvocationDoesNotExist for a brief window after
			// SendCommand returns; retry until either the invocation lands
			// or pollLimit is reached.
			if isInvocationNotFound(err) && attempts+1 < d.pollLimit() {
				continue
			}
			return "", err
		}
		switch invocation.Status {
		case "Pending", "InProgress", "Delayed":
			continue
		case "Success":
			return invocation.StandardOutputContent, nil
		case "":
			continue
		default:
			if invocation.StandardErrorContent != "" {
				return invocation.StandardOutputContent, fmt.Errorf("aws ssm: command status %s: %s", invocation.Status, invocation.StandardErrorContent)
			} else {
				return invocation.StandardOutputContent, fmt.Errorf("aws ssm: command status %s", invocation.Status)
			}
		}
	}
	return "", fmt.Errorf("aws ssm: invocation did not complete after %d polls", d.pollLimit())
}

func resolveDocumentName(osType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(osType)) {
	case "", "linux":
		return api.SSMDocumentLinux, true
	case "windows":
		return api.SSMDocumentWindows, true
	}
	return "", false
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

func isInvocationNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Code == "InvocationDoesNotExist" {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invocationdoesnotexist") ||
		strings.Contains(msg, "invocation does not exist")
}

func (d *Driver) ResolveInstanceContext(instanceID string) (region, osType string, ok bool) {
	for _, host := range GetCacheHostList() {
		if host.ID == instanceID || host.HostName == instanceID {
			return host.Region, hostOSType(host), true
		}
	}
	return "", "", false
}

func hostOSType(host schema.Host) string {
	if host.OSType != "" {
		return host.OSType
	}
	return "linux"
}

// Close ensures the host cache is flushed when callers no longer need it
// (e.g. across a session reset).
func Close() {
	SetCacheHostList(nil)
}

// errCacheMissing is exported so consumers can match on it without importing
// fmt-formatted strings.
var errCacheMissing = fmt.Errorf("aws ssm: host metadata cache miss")

// CacheMissingError exposes the sentinel for tests / callers that need to
// distinguish "no metadata" from a real API error.
func CacheMissingError() error { return errCacheMissing }
