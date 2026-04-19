package api

const ResourcesAPIVersion = "2021-04-01"

type ResourceGroup struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

type ListResourceGroupsResponse struct {
	Value    []ResourceGroup `json:"value"`
	NextLink string          `json:"nextLink"`
}
