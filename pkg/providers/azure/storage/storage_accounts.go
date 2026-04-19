package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Client          *azapi.Client
	SubscriptionIDs []string
}

func (d *Driver) GetStorages(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	logger.Info("List Storage Accounts ...")

	for _, subscription := range d.SubscriptionIDs {
		pager := azapi.NewPager[azapi.StorageAccount](d.Client, azapi.Request{
			Method: http.MethodGet,
			Path:   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Storage/storageAccounts", subscription),
			Query:  url.Values{"api-version": {azapi.StorageAPIVersion}},
			Idempotent: true,
		})
		accounts, err := pager.All(ctx)
		if err != nil {
			logger.Error("List accounts failed.")
			return list, err
		}
		for _, account := range accounts {
			accountID, err := azapi.ParseResourceID(account.ID)
			if err != nil {
				logger.Error("Parse resource ID failed.")
				return list, err
			}

			blobServices := d.GetBlobService(ctx, subscription, accountID.ResourceGroup, account.Name)
			for _, service := range blobServices {
				list = append(list, schema.Storage{
					AccountName: account.Name,
					Region:      account.Location,
					BucketName:  service + "(Blob Service)",
				})
			}

			blobContainers := d.GetBlobContainer(ctx, subscription, accountID.ResourceGroup, account.Name)
			for _, container := range blobContainers {
				list = append(list, schema.Storage{
					AccountName: account.Name,
					Region:      account.Location,
					BucketName:  container + "(Blob Container)",
				})
			}
		}
	}
	return list, nil
}
