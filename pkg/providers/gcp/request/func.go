package request

import (
	"errors"
	"fmt"

	"github.com/tidwall/gjson"
)

func (r *DefaultHttpRequest) ListManagedZones(project string) ([]string, error) {
	var zones []string
	r.Path = fmt.Sprintf("/dns/v1/projects/%s/managedZones", project)

	body, err := r.DoGetRequest()
	if err != nil {
		return zones, err
	}
	items := gjson.Get(string(body), "managedZones").Array()
	if len(items) == 0 {
		err = errors.New(gjson.Get(string(body), "error.status").String())
	}
	for _, i := range items {
		zones = append(zones, i.Get("name").String())
	}
	return zones, err
}

func (r *DefaultHttpRequest) ListRRSets(project, zone string) ([]gjson.Result, error) {
	r.Path = fmt.Sprintf("/dns/v1/projects/%s/managedZones/%s/rrsets", project, zone)

	body, err := r.DoGetRequest()
	if err != nil {
		return nil, err
	}
	items := gjson.Get(string(body), "rrsets").Array()
	if len(items) == 0 {
		err = errors.New(gjson.Get(string(body), "error.status").String())
	}

	return items, err
}

func (r *DefaultHttpRequest) ListZones(project string) ([]string, error) {
	var zones []string
	r.Path = fmt.Sprintf("/compute/v1/projects/%s/zones", project)

	body, err := r.DoGetRequest()
	if err != nil {
		return zones, err
	}
	items := gjson.Get(string(body), "items").Array()
	if len(items) == 0 {
		err = errors.New(gjson.Get(string(body), "error.status").String())
	}
	for _, i := range items {
		zones = append(zones, i.Get("name").String())
	}
	return zones, err
}

func (r *DefaultHttpRequest) ListInstances(project, zone string) ([]gjson.Result, error) {
	r.Path = fmt.Sprintf("/compute/v1/projects/%s/zones/%s/instances", project, zone)

	body, err := r.DoGetRequest()
	if err != nil {
		return nil, err
	}
	items := gjson.Get(string(body), "items").Array()
	if len(items) == 0 {
		err = errors.New(gjson.Get(string(body), "error.status").String())
	}

	return items, err
}

func (r *DefaultHttpRequest) ListServiceAccounts(project string) (map[string]string, error) {
	accounts := make(map[string]string)
	r.Path = fmt.Sprintf("/v1/projects/%s/serviceAccounts", project)

	body, err := r.DoGetRequest()
	if err != nil {
		return accounts, err
	}
	items := gjson.Get(string(body), "accounts").Array()
	if len(items) == 0 {
		err = errors.New(gjson.Get(string(body), "error.status").String())
	}
	for _, i := range items {
		name := i.Get("displayName").String()
		accounts[name] = i.Get("uniqueId").String()
	}
	return accounts, err
}
