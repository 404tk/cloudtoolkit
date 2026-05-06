package api

const StorageBaseURL = "https://storage.googleapis.com"

type GCSBucket struct {
	Kind         string `json:"kind"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	StorageClass string `json:"storageClass"`
	Location     string `json:"location"`
	TimeCreated  string `json:"timeCreated"`
}

type GCSBucketsListResponse struct {
	Items         []GCSBucket `json:"items"`
	NextPageToken string      `json:"nextPageToken"`
}

type GCSObject struct {
	Kind         string `json:"kind"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	Bucket       string `json:"bucket"`
	StorageClass string `json:"storageClass"`
	Size         string `json:"size"`
	Updated      string `json:"updated"`
	TimeCreated  string `json:"timeCreated"`
}

type GCSObjectsListResponse struct {
	Items         []GCSObject `json:"items"`
	Prefixes      []string    `json:"prefixes"`
	NextPageToken string      `json:"nextPageToken"`
}

// GCSPolicy mirrors the IAM policy returned by the GCS bucket
// `getIamPolicy` action and accepted by `setIamPolicy`.
type GCSPolicy struct {
	Version  int             `json:"version"`
	Bindings []GCSPolicyBind `json:"bindings"`
	Etag     string          `json:"etag"`
}

type GCSPolicyBind struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}
