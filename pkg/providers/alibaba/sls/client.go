package sls

import (
	"fmt"
	"net/http"
	"os"
)

type Client struct {
	accessKeyId     string //Access Key Id
	accessKeySecret string //Access Key Secret
	securityToken   string //sts token
	debug           bool
	httpClient      *http.Client
	version         string
	internal        bool
	region          string
	endpoint        string
}

func (client *Client) SetDebug(debug bool) {
	client.debug = debug
}

// SetTransport sets transport to the http client
func (client *Client) SetTransport(transport http.RoundTripper) {
	if client.httpClient == nil {
		client.httpClient = &http.Client{}
	}
	client.httpClient.Transport = transport
}

const (
	SLSDefaultEndpoint = "log.aliyuncs.com"
	SLSAPIVersion      = "0.6.0"
	Version            = "0.0.1"
)

// NewClient creates a new instance of ECS client
func NewClient(internal bool, region, accessKeyId, accessKeySecret, securityToken string) *Client {
	endpoint := os.Getenv("SLS_ENDPOINT")
	if endpoint == "" {
		endpoint = SLSDefaultEndpoint
	}
	return NewClientWithEndpoint(endpoint, region, internal, accessKeyId, accessKeySecret, securityToken)
}

func NewClientForAssumeRole(internal bool, region, accessKeyId, accessKeySecret, securityToken string) *Client {
	endpoint := os.Getenv("SLS_ENDPOINT")
	if endpoint == "" {
		endpoint = SLSDefaultEndpoint
	}

	return &Client{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
		securityToken:   securityToken,
		internal:        internal,
		region:          region,
		version:         SLSAPIVersion,
		endpoint:        endpoint,
		httpClient:      &http.Client{},
	}
}

func NewClientWithEndpoint(endpoint string, region string, internal bool, accessKeyId, accessKeySecret, securityToken string) *Client {
	return &Client{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
		securityToken:   securityToken,
		internal:        internal,
		region:          region,
		version:         SLSAPIVersion,
		endpoint:        endpoint,
		httpClient:      &http.Client{},
	}
}

func (client *Client) forProject(name string) *Client {
	newclient := *client

	region := string(client.region)
	if client.internal {
		region = fmt.Sprintf("%s-intranet", region)
	}
	if name == "" {
		newclient.endpoint = fmt.Sprintf("%s.%s", region, client.endpoint)
	} else {
		newclient.endpoint = fmt.Sprintf("%s.%s.%s", name, region, client.endpoint)
	}
	return &newclient
}

type Project struct {
	CreateTime      *string `json:"createTime,omitempty" xml:"createTime,omitempty"`
	Description     *string `json:"description,omitempty" xml:"description,omitempty"`
	LastModifyTime  *string `json:"lastModifyTime,omitempty" xml:"lastModifyTime,omitempty"`
	Owner           *string `json:"owner,omitempty" xml:"owner,omitempty"`
	ProjectName     *string `json:"projectName,omitempty" xml:"projectName,omitempty"`
	Region          *string `json:"region,omitempty" xml:"region,omitempty"`
	ResourceGroupId *string `json:"resourceGroupId,omitempty" xml:"resourceGroupId,omitempty"`
	Status          *string `json:"status,omitempty" xml:"status,omitempty"`
}

type ListProjectRequest struct {
	Offset      int32  `json:"offset,omitempty" xml:"offset,omitempty"`
	ProjectName string `json:"projectName,omitempty" xml:"projectName,omitempty"`
	Size        int32  `json:"size,omitempty" xml:"size,omitempty"`
}

func (r ListProjectRequest) Map() map[string]string {
	m := map[string]string{
		"offset": fmt.Sprintf("%v", r.Offset),
	}
	if r.ProjectName != "" {
		m["projectName"] = r.ProjectName
	}
	if r.Size == 0 {
		m["size"] = "100" // default 100
	} else {
		m["size"] = fmt.Sprintf("%v", r.Size)
	}
	return m
}

type ListProjectResponse struct {
	Count    *int64     `json:"count,omitempty" xml:"count,omitempty"`
	Projects []*Project `json:"projects,omitempty" xml:"projects,omitempty" type:"Repeated"`
	Total    *int64     `json:"total,omitempty" xml:"total,omitempty"`
}

func (client *Client) ListProject(r ListProjectRequest) (*ListProjectResponse, error) {
	req := &request{
		method: "GET",
		path:   "/",
		params: r.Map(),
	}

	newClient := client.forProject("")
	resp := &ListProjectResponse{}
	err := newClient.requestWithJsonResponse(req, resp)
	return resp, err
}
