package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) GetBlobContainer(ctx context.Context, subscription, groupName, accountName string) []string {
	pager := azapi.NewPager[azapi.BlobContainer](d.Client, azapi.Request{
		Method: http.MethodGet,
		Path: fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/default/containers",
			subscription,
			groupName,
			accountName,
		),
		Query:      url.Values{"api-version": {azapi.StorageAPIVersion}},
		Idempotent: true,
	})
	items, err := pager.All(ctx)
	if err != nil {
		logger.Error("List blob containers failed:", err.Error())
		return nil
	}

	blobs := make([]string, 0, len(items))
	for _, blob := range items {
		if blob.Name != "" {
			blobs = append(blobs, blob.Name)
		}
	}
	return blobs
}

// ContainerInfo carries the auditable fields for a blob container.
type ContainerInfo struct {
	Subscription  string
	ResourceGroup string
	AccountName   string
	Name          string
	PublicAccess  string
}

// ListBlobContainers returns containers across every configured subscription
// + storage account. It is the read-side helper backing the bucket-acl-check
// `audit` action.
func (d *Driver) ListBlobContainers(ctx context.Context) ([]ContainerInfo, error) {
	if d == nil || d.Client == nil {
		return nil, fmt.Errorf("azure storage: nil client")
	}
	out := make([]ContainerInfo, 0)
	for _, subscription := range d.SubscriptionIDs {
		accountsPager := azapi.NewPager[azapi.StorageAccount](d.Client, azapi.Request{
			Method:     http.MethodGet,
			Path:       fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Storage/storageAccounts", subscription),
			Query:      url.Values{"api-version": {azapi.StorageAPIVersion}},
			Idempotent: true,
		})
		accounts, err := accountsPager.All(ctx)
		if err != nil {
			return out, err
		}
		for _, account := range accounts {
			parsed, err := azapi.ParseResourceID(account.ID)
			if err != nil {
				return out, err
			}
			containersPager := azapi.NewPager[azapi.BlobContainer](d.Client, azapi.Request{
				Method: http.MethodGet,
				Path: fmt.Sprintf(
					"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/default/containers",
					subscription, parsed.ResourceGroup, account.Name,
				),
				Query:      url.Values{"api-version": {azapi.StorageAPIVersion}},
				Idempotent: true,
			})
			containers, err := containersPager.All(ctx)
			if err != nil {
				return out, err
			}
			for _, container := range containers {
				level := "None"
				if container.Properties != nil && container.Properties.PublicAccess != "" {
					level = container.Properties.PublicAccess
				}
				out = append(out, ContainerInfo{
					Subscription:  subscription,
					ResourceGroup: parsed.ResourceGroup,
					AccountName:   account.Name,
					Name:          container.Name,
					PublicAccess:  level,
				})
			}
		}
	}
	return out, nil
}

// FindContainer locates a container by name across every configured
// subscription. Returns the first match.
func (d *Driver) FindContainer(ctx context.Context, name string) (ContainerInfo, error) {
	containers, err := d.ListBlobContainers(ctx)
	if err != nil {
		return ContainerInfo{}, err
	}
	for _, c := range containers {
		if c.Name == name {
			return c, nil
		}
	}
	return ContainerInfo{}, fmt.Errorf("container %q not found", name)
}

// GetContainerACL fetches the publicAccess property for a container.
func (d *Driver) GetContainerACL(ctx context.Context, subscription, group, account, container string) (string, error) {
	if d == nil || d.Client == nil {
		return "", fmt.Errorf("azure storage: nil client")
	}
	var result azapi.BlobContainer
	if err := d.Client.Do(ctx, azapi.Request{
		Method: http.MethodGet,
		Path: fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/default/containers/%s",
			subscription, group, account, container,
		),
		Query:      url.Values{"api-version": {azapi.StorageAPIVersion}},
		Idempotent: true,
	}, &result); err != nil {
		return "", err
	}
	if result.Properties == nil || result.Properties.PublicAccess == "" {
		return "None", nil
	}
	return result.Properties.PublicAccess, nil
}

// SetContainerACL updates the publicAccess property for a container. `level`
// is normalized to one of "None" / "Blob" / "Container".
func (d *Driver) SetContainerACL(ctx context.Context, subscription, group, account, container, level string) error {
	if d == nil || d.Client == nil {
		return fmt.Errorf("azure storage: nil client")
	}
	normalized, err := normalizePublicAccess(level)
	if err != nil {
		return err
	}
	body, err := json.Marshal(azapi.BlobContainerPatchRequest{
		Properties: azapi.BlobContainerProperties{PublicAccess: normalized},
	})
	if err != nil {
		return err
	}
	return d.Client.Do(ctx, azapi.Request{
		Method: http.MethodPatch,
		Path: fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/default/containers/%s",
			subscription, group, account, container,
		),
		Query: url.Values{"api-version": {azapi.StorageAPIVersion}},
		Body:  body,
	}, nil)
}

func normalizePublicAccess(level string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "none":
		return "None", nil
	case "blob":
		return "Blob", nil
	case "container":
		return "Container", nil
	}
	return "", fmt.Errorf("azure storage: unsupported publicAccess level %q (expected None, Blob, or Container)", level)
}
