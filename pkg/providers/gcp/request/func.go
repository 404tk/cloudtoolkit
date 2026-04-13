package request

import (
	"encoding/json"
	"errors"
	"fmt"
)

type ManagedZone struct {
	Name    string `json:"name"`
	DNSName string `json:"dnsName"`
}

type RRSet struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	RRDatas []string `json:"rrdatas"`
}

type Instance struct {
	Hostname          string             `json:"hostname"`
	Zone              string             `json:"zone"`
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces"`
}

type NetworkInterface struct {
	NetworkIP     string         `json:"networkIP"`
	AccessConfigs []AccessConfig `json:"accessConfigs"`
}

type AccessConfig struct {
	NatIP string `json:"natIP"`
}

type ServiceAccount struct {
	DisplayName string `json:"displayName"`
	UniqueID    string `json:"uniqueId"`
}

type listManagedZonesResponse struct {
	ManagedZones []ManagedZone `json:"managedZones"`
	Error        *apiErrorBody `json:"error"`
}

type listRRSetsResponse struct {
	RRSets []RRSet       `json:"rrsets"`
	Error  *apiErrorBody `json:"error"`
}

type listZonesResponse struct {
	Items []struct {
		Name string `json:"name"`
	} `json:"items"`
	Error *apiErrorBody `json:"error"`
}

type listInstancesResponse struct {
	Items []Instance    `json:"items"`
	Error *apiErrorBody `json:"error"`
}

type listServiceAccountsResponse struct {
	Accounts []ServiceAccount `json:"accounts"`
	Error    *apiErrorBody    `json:"error"`
}

type apiErrorBody struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (e *apiErrorBody) Err() error {
	if e == nil {
		return nil
	}
	switch {
	case e.Message != "":
		return errors.New(e.Message)
	case e.Status != "":
		return errors.New(e.Status)
	default:
		return errors.New("request failed")
	}
}

func (r *DefaultHttpRequest) ListManagedZones(project string) ([]ManagedZone, error) {
	r.Path = fmt.Sprintf("/dns/v1/projects/%s/managedZones", project)

	body, err := r.DoGetRequest()
	if err != nil {
		return nil, err
	}

	var resp listManagedZonesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if err := resp.Error.Err(); err != nil {
		return nil, err
	}
	return resp.ManagedZones, nil
}

func (r *DefaultHttpRequest) ListRRSets(project, zone string) ([]RRSet, error) {
	r.Path = fmt.Sprintf("/dns/v1/projects/%s/managedZones/%s/rrsets", project, zone)

	body, err := r.DoGetRequest()
	if err != nil {
		return nil, err
	}

	var resp listRRSetsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if err := resp.Error.Err(); err != nil {
		return nil, err
	}
	return resp.RRSets, nil
}

func (r *DefaultHttpRequest) ListZones(project string) ([]string, error) {
	r.Path = fmt.Sprintf("/compute/v1/projects/%s/zones", project)

	body, err := r.DoGetRequest()
	if err != nil {
		return nil, err
	}

	var resp listZonesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if err := resp.Error.Err(); err != nil {
		return nil, err
	}

	zones := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item.Name != "" {
			zones = append(zones, item.Name)
		}
	}
	return zones, nil
}

// https://cloud.google.com/compute/docs/reference/rest/v1/instances/list
func (r *DefaultHttpRequest) ListInstances(project, zone string) ([]Instance, error) {
	r.Path = fmt.Sprintf("/compute/v1/projects/%s/zones/%s/instances", project, zone)

	body, err := r.DoGetRequest()
	if err != nil {
		return nil, err
	}

	var resp listInstancesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if err := resp.Error.Err(); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (r *DefaultHttpRequest) ListServiceAccounts(project string) (map[string]string, error) {
	r.Path = fmt.Sprintf("/v1/projects/%s/serviceAccounts", project)

	body, err := r.DoGetRequest()
	if err != nil {
		return nil, err
	}

	var resp listServiceAccountsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if err := resp.Error.Err(); err != nil {
		return nil, err
	}

	accounts := make(map[string]string, len(resp.Accounts))
	for _, account := range resp.Accounts {
		if account.DisplayName == "" {
			continue
		}
		accounts[account.DisplayName] = account.UniqueID
	}
	return accounts, nil
}
