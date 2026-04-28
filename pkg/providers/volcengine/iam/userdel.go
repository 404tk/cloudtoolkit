package iam

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) DelUser() (schema.IAMResult, error) {
	ctx := context.Background()
	client, err := d.requireClient()
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("require client failed: %w", err)
	}

	region := d.requestRegion()
	userName := strings.TrimSpace(d.UserName)
	if userName == "" {
		return schema.IAMResult{}, fmt.Errorf("empty user name")
	}

	if err := detachUserPolicy(ctx, client, region, userName); err != nil && !isIgnorableDeleteError(err) {
		return schema.IAMResult{}, fmt.Errorf("remove policy from %s failed: %w", userName, err)
	}
	if err := deleteLoginProfile(ctx, client, region, userName); err != nil && !isIgnorableDeleteError(err) {
		return schema.IAMResult{}, fmt.Errorf("delete login profile for %s failed: %w", userName, err)
	}
	if err := deleteUser(ctx, client, region, userName); err != nil {
		return schema.IAMResult{}, fmt.Errorf("delete user %s failed: %w", userName, err)
	}

	return schema.IAMResult{
		Username: userName,
		Message:  "User deleted successfully",
	}, nil
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
