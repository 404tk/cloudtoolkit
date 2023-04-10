package sms

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
)

type SmsProvider struct {
	Client *dysmsapi.Client
}

func (d *SmsProvider) GetResource(ctx context.Context) (schema.Sms, error) {
	res := schema.Sms{}
	select {
	case <-ctx.Done():
		return res, nil
	default:
		log.Println("[*] List SMS resource ...")
	}
	var err error
	res.Signs, err = listSmsSign(d.Client)
	if err != nil {
		log.Println("[-] List SMS failed.")
		return res, err
	}
	res.Templates, err = listSmsTemplate(d.Client)
	res.DailySize, err = querySendStatistics(d.Client)

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
