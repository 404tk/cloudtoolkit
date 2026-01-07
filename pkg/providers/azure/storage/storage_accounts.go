package storage

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/storage/mgmt/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

type Driver struct {
	SubscriptionIDs []string
	Authorizer      autorest.Authorizer
}

func (d *Driver) GetStorages(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	logger.Info("List Storage Accounts ...")
	for _, subscription := range d.SubscriptionIDs {
		accountsClient := storage.NewAccountsClient(subscription)
		accountsClient.Authorizer = d.Authorizer
		accounts, err := accountsClient.List(ctx)
		if err != nil {
			logger.Error("List accounts failed.")
			return list, err
		}
		for _, account := range accounts.Values() {
			accountId, err := azure.ParseResourceID(*account.ID)
			if err != nil {
				logger.Error("Parse resource ID failed.")
				return list, err
			}

			blobService := d.GetBlobService(ctx, subscription, accountId.ResourceGroup, *account.Name)
			for _, s := range blobService {
				_account := schema.Storage{
					AccountName: *account.Name,
					Region:      *account.Location,
					BucketName:  s + "(Blob Service)",
				}
				list = append(list, _account)
			}

			blobContainer := d.GetBlobContainer(ctx, subscription, accountId.ResourceGroup, *account.Name)
			for _, c := range blobContainer {
				_account := schema.Storage{
					AccountName: *account.Name,
					Region:      *account.Location,
					BucketName:  c + "(Blob Container)",
				}
				list = append(list, _account)
			}
		}
	}
	return list, nil
}
