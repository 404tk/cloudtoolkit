package iam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type DefaultHttpRequest struct {
	Endpoint string
	Path     string
	Method   string

	QueryParams map[string]interface{}
	// pathParams   map[string]string
	HeaderParams map[string]string
	// formParams   map[string]def.FormData
	Body []byte

	// autoFilledPathParams map[string]string
}

func (httpRequest *DefaultHttpRequest) DoGetRequest(auth, timestamp string) ([]byte, error) {
	url := "https://" + httpRequest.Endpoint + httpRequest.Path
	if len(httpRequest.QueryParams) > 0 {
		url += "?" + CanonicalQueryString(httpRequest)
	}
	req, err := http.NewRequest(httpRequest.Method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", auth)
	req.Header.Add("X-Sdk-Date", timestamp)

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	return body, err
}

func (httpRequest *DefaultHttpRequest) CanonicalSliceQueryParamsToMulti(value reflect.Value) []string {
	params := make([]string, 0)

	for i := 0; i < value.Len(); i++ {
		if value.Index(i).Kind() == reflect.Struct {
			v, e := json.Marshal(value.Interface())
			if e == nil {
				if strings.HasPrefix(string(v), "\"") {
					params = append(params, strings.Trim(string(v), "\""))
				} else {
					params = append(params, string(v))
				}
			}
		} else {
			params = append(params, fmt.Sprintf("%v", value.Index(i)))
		}
	}

	return params
}

func (httpRequest *DefaultHttpRequest) CanonicalMapQueryParams(key string, value reflect.Value) []map[string]string {
	queryParams := make([]map[string]string, 0)

	for _, k := range value.MapKeys() {
		if value.MapIndex(k).Kind() == reflect.Struct {
			v, e := json.Marshal(value.Interface())
			if e == nil {
				if strings.HasPrefix(string(v), "\"") {
					queryParams = append(queryParams, map[string]string{
						key: strings.Trim(string(v), "\""),
					})
				} else {
					queryParams = append(queryParams, map[string]string{
						key: string(v),
					})
				}
			}
		} else if value.MapIndex(k).Kind() == reflect.Slice {
			params := httpRequest.CanonicalSliceQueryParamsToMulti(value.MapIndex(k))
			if len(params) == 0 {
				queryParams = append(queryParams, map[string]string{
					fmt.Sprintf("%s[%s]", key, k): "",
				})
				continue
			}
			for _, paramValue := range httpRequest.CanonicalSliceQueryParamsToMulti(value.MapIndex(k)) {
				queryParams = append(queryParams, map[string]string{
					fmt.Sprintf("%s[%s]", key, k): paramValue,
				})
			}
		} else if value.MapIndex(k).Kind() == reflect.Map {
			queryParams = append(queryParams, httpRequest.CanonicalMapQueryParams(fmt.Sprintf("%s[%s]", key, k), value.MapIndex(k))...)
		} else {
			queryParams = append(queryParams, map[string]string{
				fmt.Sprintf("%s[%s]", key, k): fmt.Sprintf("%v", value.MapIndex(k)),
			})
		}
	}

	return queryParams
}
