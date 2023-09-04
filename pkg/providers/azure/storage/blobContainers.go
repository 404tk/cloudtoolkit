package storage

import (
	"context"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/storage/mgmt/storage"
)

func (d *Driver) GetBlobContainer(ctx context.Context, subscription, groupName, accountName string) []string {
	var blobs []string
	client := storage.NewBlobContainersClient(subscription)
	client.Authorizer = d.Authorizer

	resp, err := client.List(context.Background(), groupName, accountName, "", "", "")
	if err != nil {
		logger.Error("List blob containers failed:", err.Error())
		return blobs
	}
	for _, blob := range resp.Values() {
		blobs = append(blobs, *blob.Name)
	}

	return blobs
}
