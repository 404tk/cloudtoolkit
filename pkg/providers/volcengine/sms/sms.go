// Package sms wraps Volcengine SMS for the cloudlist `sms` asset.
// Pattern-inferred — see api/types_sms.go.
package sms

import (
	"context"
	"errors"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	defaultRegion = "cn-beijing"
	pageSize      = 100
	maxPages      = 50
)

type Driver struct {
	Client *api.Client
	Region string
}

func (d *Driver) requestRegion() string {
	if d.Region != "" && d.Region != "all" {
		return d.Region
	}
	return defaultRegion
}

func (d *Driver) GetResource(ctx context.Context) (schema.Sms, error) {
	out := schema.Sms{}
	if d == nil || d.Client == nil {
		return out, errors.New("volcengine sms: nil api client")
	}
	logger.Info("List Volcengine SMS signs and templates ...")
	region := d.requestRegion()

	for page := 1; page <= maxPages; page++ {
		resp, err := d.Client.ListSmsSigns(ctx, region, page, pageSize)
		if err != nil {
			return out, err
		}
		for _, s := range resp.Result.List {
			out.Signs = append(out.Signs, schema.SmsSign{
				Name:   s.Sign,
				Type:   s.SignType,
				Status: firstNonEmpty(s.Status, s.Reason),
			})
		}
		if len(resp.Result.List) < pageSize {
			break
		}
	}

	for page := 1; page <= maxPages; page++ {
		resp, err := d.Client.ListSmsTemplates(ctx, region, page, pageSize)
		if err != nil {
			return out, err
		}
		for _, t := range resp.Result.List {
			out.Templates = append(out.Templates, schema.SmsTemplate{
				Name:    firstNonEmpty(t.TemplateName, t.TemplateID),
				Status:  firstNonEmpty(t.Status, t.Reason),
				Content: t.Content,
			})
		}
		if len(resp.Result.List) < pageSize {
			break
		}
	}

	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
