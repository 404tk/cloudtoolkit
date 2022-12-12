package storage

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/storage/mgmt/storage"
)

func (d *StorageAccountProvider) GetBlobService(ctx context.Context, subscription, groupName, accountName string) []string {
	var blobs []string
	client := storage.NewBlobServicesClient(subscription)
	client.Authorizer = d.Authorizer

	resp, err := client.List(context.Background(), groupName, accountName)
	if err != nil {
		log.Println("[-] List blob services failed:", err.Error())
		return blobs
	}
	for _, blob := range *resp.Value {
		blobs = append(blobs, *blob.Name)
	}

	return blobs
}
