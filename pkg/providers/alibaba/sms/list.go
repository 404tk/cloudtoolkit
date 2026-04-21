package sms

import (
	"context"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Cred          aliauth.Credential
	Region        string
	clientOptions []api.Option
	now           func() time.Time
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) GetResource(ctx context.Context) (schema.Sms, error) {
	res := schema.Sms{}
	select {
	case <-ctx.Done():
		return res, nil
	default:
		logger.Info("List SMS resource ...")
	}
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	client := api.NewClient(d.Cred, d.clientOptions...)
	region = api.NormalizeRegion(region)
	var err error
	res.Signs, err = listSmsSign(ctx, client, region)
	if err != nil {
		logger.Error("List SMS failed.")
		return res, err
	}
	res.Templates, _ = listSmsTemplate(ctx, client, region)
	res.DailySize, _ = d.querySendStatistics(ctx, client, region)

	return res, err
}

var status = map[string]string{
	"AUDIT_STATE_INIT":     "审核中",
	"AUDIT_STATE_PASS":     "审核通过",
	"AUDIT_STATE_NOT_PASS": "审核未通过",
	"AUDIT_STATE_CANCEL":   "取消审核",
}

func listSmsSign(ctx context.Context, client *api.Client, region string) ([]schema.SmsSign, error) {
	signs := []schema.SmsSign{}
	response, err := client.QuerySMSSignList(ctx, region)
	if err != nil {
		return signs, err
	}
	for _, sign := range response.SmsSignList {
		signs = append(signs, schema.SmsSign{
			Name:   sign.SignName,
			Type:   sign.BusinessType,
			Status: status[sign.AuditStatus],
		})
	}
	return signs, nil
}

func listSmsTemplate(ctx context.Context, client *api.Client, region string) ([]schema.SmsTemplate, error) {
	temps := []schema.SmsTemplate{}
	response, err := client.QuerySMSTemplateList(ctx, region)
	if err != nil {
		return temps, err
	}
	for _, temp := range response.SmsTemplateList {
		s, ok := status[temp.AuditStatus]
		if !ok {
			continue
		}
		temps = append(temps, schema.SmsTemplate{
			Name:    temp.TemplateName,
			Status:  s,
			Content: temp.TemplateContent,
		})
	}
	return temps, nil
}
