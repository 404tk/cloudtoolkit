package storage

import (
	"context"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/storage/mgmt/storage"
)

func (d *Driver) GetBlobService(ctx context.Context, subscription, groupName, accountName string) []string {
	var blobs []string
	client := storage.NewBlobServicesClient(subscription)
	client.Authorizer = d.Authorizer

	resp, err := client.List(context.Background(), groupName, accountName)
	if err != nil {
		logger.Error("List blob services failed:", err.Error())
		return blobs
	}
	for _, blob := range *resp.Value {
		blobs = append(blobs, *blob.Name)
	}

	return blobs
}
