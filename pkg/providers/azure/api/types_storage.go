package api

const StorageAPIVersion = "2022-05-01"

type StorageAccount struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

type ListStorageAccountsResponse struct {
	Value    []StorageAccount `json:"value"`
	NextLink string           `json:"nextLink"`
}

type BlobService struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListBlobServicesResponse struct {
	Value []BlobService `json:"value"`
}

type BlobContainer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListBlobContainersResponse struct {
	Value    []BlobContainer `json:"value"`
	NextLink string          `json:"nextLink"`
}
