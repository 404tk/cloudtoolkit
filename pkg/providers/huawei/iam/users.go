package iam

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/tidwall/gjson"
)

type IAMUserProvider struct {
	Auth    basic.Credentials
	Regions []string
}

func (d *IAMUserProvider) GetIAMUser(ctx context.Context) ([]*schema.User, error) {
	list := schema.NewResources().Users
	log.Println("[*] Start enumerating IAM user ...")
	r := NewGetRequest()
	users, err := r.ListUsers(d.Auth.AK, d.Auth.SK)
	if err != nil {
		log.Println("[-] Enumerate IAM failed.")
		return list, err
	}

	for _, u := range users {
		_user := schema.User{
			UserName:    u.Get("name").String(),
			UserId:      u.Get("id").String(),
			EnableLogin: u.Get("enabled").Bool(),
		}
		list = append(list, &_user)
	}

	return list, nil
}

func (r *DefaultHttpRequest) ListUsers(accesskey, secretkey string) ([]gjson.Result, error) {
	r.Path = "/v3/users"
	// r.QueryParams = map[string]interface{}{"enabled": reflect.ValueOf("true")}
	auth, err := Sign(r, accesskey, secretkey)
	if err != nil {
		return nil, err
	}
	body, err := r.DoGetRequest(auth["Authorization"], r.HeaderParams["X-Sdk-Date"])
	if err != nil {
		return nil, err
	}
	/*
		if strings.Contains(string(body),"Forbidden"){
			return nil,errors.New("You are not authorized to perform the requested action.")
		}
	*/
	users := gjson.Get(string(body), "users").Array()
	return users, err
}
