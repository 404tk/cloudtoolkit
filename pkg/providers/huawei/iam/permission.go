package iam

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type getUserIDResponse struct {
	Credential struct {
		UserID string `json:"user_id"`
	} `json:"credential"`
	ErrorMsg string `json:"error_msg"`
}

type getUserNameResponse struct {
	User struct {
		Name string `json:"name"`
	} `json:"user"`
}

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

	var resp getUserIDResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	userID := resp.Credential.UserID
	if userID == "" {
		err = errors.New(resp.ErrorMsg)
	}
	return userID, err
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

	var resp getUserNameResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	return resp.User.Name, err
}
