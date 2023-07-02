package sms

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
)

type Driver struct {
	Cred   *credentials.StsTokenCredential
	Region string
}

func (d *Driver) GetResource(ctx context.Context) (schema.Sms, error) {
	res := schema.Sms{}
	select {
	case <-ctx.Done():
		return res, nil
	default:
		log.Println("[*] List SMS resource ...")
	}
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	client, err := dysmsapi.NewClientWithOptions(region, sdk.NewConfig(), d.Cred)
	if err != nil {
		return res, err
	}
	res.Signs, err = listSmsSign(client)
	if err != nil {
		log.Println("[-] List SMS failed.")
		return res, err
	}
	res.Templates, err = listSmsTemplate(client)
	res.DailySize, err = querySendStatistics(client)

	return res, err
}

var status = map[string]string{
	"AUDIT_STATE_INIT":     "审核中",
	"AUDIT_STATE_PASS":     "审核通过",
	"AUDIT_STATE_NOT_PASS": "审核未通过",
	"AUDIT_STATE_CANCEL":   "取消审核",
}

func listSmsSign(client *dysmsapi.Client) ([]schema.SmsSign, error) {
	signs := []schema.SmsSign{}
	request := dysmsapi.CreateQuerySmsSignListRequest()
	request.Scheme = "https"
	response, err := client.QuerySmsSignList(request)
	if err != nil {
		return signs, err
	}
	for _, sign := range response.SmsSignList {
		s, _ := status[sign.AuditStatus]
		signs = append(signs, schema.SmsSign{
			Name:   sign.SignName,
			Type:   sign.BusinessType,
			Status: s,
		})
	}
	return signs, nil
}

func listSmsTemplate(client *dysmsapi.Client) ([]schema.SmsTemplate, error) {
	temps := []schema.SmsTemplate{}
	request := dysmsapi.CreateQuerySmsTemplateListRequest()
	request.Scheme = "https"
	response, err := client.QuerySmsTemplateList(request)
	if err != nil {
		return temps, err
	}
	for _, temp := range response.SmsTemplateList {
		s, _ := status[temp.AuditStatus]
		temps = append(temps, schema.SmsTemplate{
			Name:    temp.TemplateName,
			Status:  s,
			Content: temp.TemplateContent,
		})
	}
	return temps, nil
}
