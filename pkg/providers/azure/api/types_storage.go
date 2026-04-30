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
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Properties *BlobContainerProperties `json:"properties,omitempty"`
}

type BlobContainerProperties struct {
	PublicAccess string `json:"publicAccess,omitempty"`
}

type ListBlobContainersResponse struct {
	Value    []BlobContainer `json:"value"`
	NextLink string          `json:"nextLink"`
}

// BlobContainerPatchRequest is the body of PATCH on the container resource. It
// is also valid for PUT (update). PublicAccess values: "None", "Blob",
// "Container".
type BlobContainerPatchRequest struct {
	Properties BlobContainerProperties `json:"properties"`
}
