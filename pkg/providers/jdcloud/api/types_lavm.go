package api

import "net/url"

type DescribeLAVMInstancesResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Instances  []LAVMInstance `json:"instances"`
		TotalCount int            `json:"totalCount"`
	} `json:"result"`
}

type LAVMInstance struct {
	InstanceID       string       `json:"instanceId"`
	Status           string       `json:"status"`
	PrivateIPAddress string       `json:"innerIpAddress"`
	PublicIPAddress  string       `json:"publicIpAddress"`
	RegionID         string       `json:"regionId"`
	InstanceName     string       `json:"instanceName"`
	BusinessStatus   string       `json:"businessStatus"`
	ImageID          string       `json:"imageId"`
	Domains          []LAVMDomain `json:"domains"`
}

type LAVMDomain struct {
	DomainName string `json:"domainName"`
}

type DescribeLAVMImagesResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Images []LAVMImage `json:"images"`
	} `json:"result"`
}

type LAVMImage struct {
	ImageID  string `json:"imageId"`
	OSType   string `json:"osType"`
	Platform string `json:"platform"`
}

func NewLAVMDescribeImagesQuery(imageIDs string) url.Values {
	query := url.Values{}
	if imageIDs != "" {
		query.Set("imageIds", imageIDs)
	}
	return query
}
