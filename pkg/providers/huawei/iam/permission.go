package iam

import (
	"errors"
	"fmt"
	"time"

	"github.com/tidwall/gjson"
)

func NewGetRequest(region string) *DefaultHttpRequest {
	timestamp := time.Now().UTC().Format(BasicDateFormat)
	return &DefaultHttpRequest{
		Endpoint:     fmt.Sprintf("iam.%s.myhuaweicloud.com", region),
		Method:       "GET",
		HeaderParams: map[string]string{"X-Sdk-Date": timestamp},
	}
}

func (r *DefaultHttpRequest) GetUserId(accesskey, secretkey string) (string, error) {
	r.Path = fmt.Sprintf("/v3.0/OS-CREDENTIAL/credentials/%s", accesskey)
	auth, err := Sign(r, accesskey, secretkey)
	if err != nil {
		return "", err
	}

	body, err := r.DoGetRequest(auth["Authorization"], r.HeaderParams["X-Sdk-Date"])
	if err != nil {
		return "", err
	}
	user_id := gjson.Get(string(body), "credential.user_id").String()
	if user_id == "" {
		err = errors.New(gjson.Get(string(body), "error_msg").String())
	}
	return user_id, err
}

func (r *DefaultHttpRequest) GetUserName(accesskey, secretkey string) (string, error) {
	user_id, err := r.GetUserId(accesskey, secretkey)
	if err != nil {
		return "", err
	}
	r.Path = fmt.Sprintf("/v3/users/%s", user_id)
	auth, err := Sign(r, accesskey, secretkey)
	if err != nil {
		return "", err
	}

	body, err := r.DoGetRequest(auth["Authorization"], r.HeaderParams["X-Sdk-Date"])
	if err != nil {
		return "", err
	}
	username := gjson.Get(string(body), "user.name").String()
	return username, err
}
