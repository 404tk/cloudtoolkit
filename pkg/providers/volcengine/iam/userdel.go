package iam

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) DelUser() {
	ctx := context.Background()
	client, err := d.requireClient()
	if err != nil {
		logger.Error(err)
		return
	}

	region := d.requestRegion()
	userName := strings.TrimSpace(d.UserName)
	if userName == "" {
		logger.Error("Empty user name.")
		return
	}

	if err := detachUserPolicy(ctx, client, region, userName); err != nil && !isIgnorableDeleteError(err) {
		logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", userName, err))
		return
	}
	if err := deleteLoginProfile(ctx, client, region, userName); err != nil && !isIgnorableDeleteError(err) {
		logger.Error(fmt.Sprintf("Delete login profile for %s failed: %s", userName, err))
		return
	}
	if err := deleteUser(ctx, client, region, userName); err != nil {
		logger.Error(fmt.Sprintf("Delete user %s failed: %s", userName, err))
		return
	}
	logger.Warning(userName + " user delete completed.")
}

func detachUserPolicy(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.DetachUserPolicy(ctx, region, userName, administratorPolicyName, systemPolicyType)
	return err
}

func deleteLoginProfile(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.DeleteLoginProfile(ctx, region, userName)
	return err
}

func deleteUser(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.DeleteUser(ctx, region, userName)
	return err
}

func isIgnorableDeleteError(err error) bool {
	if err == nil {
		return false
	}

	code := strings.ToLower(strings.TrimSpace(api.ErrorCode(err)))
	if code != "" {
		switch {
		case strings.Contains(code, "notexist"),
			strings.Contains(code, "notfound"),
			strings.Contains(code, "entitynotexist"),
			strings.Contains(code, "alreadydetached"):
			return true
		}
	}

	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		message := strings.ToLower(strings.TrimSpace(apiErr.Message))
		switch {
		case strings.Contains(message, "not exist"),
			strings.Contains(message, "not found"),
			strings.Contains(message, "already detached"):
			return true
		}
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not exist") ||
		strings.Contains(message, "not found") ||
		strings.Contains(message, "already detached")
}
