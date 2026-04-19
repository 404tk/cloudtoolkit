package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

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
