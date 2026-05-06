// Package msgsms wraps Huawei Cloud MSGSMS template + sign listing for the
// cloudlist `sms` asset.
//
// Pattern-inferred — see api/types_msgsms.go.
package msgsms

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const defaultRegion = "cn-north-4"

type Driver struct {
	Cred    auth.Credential
	Regions []string
	Client  *api.Client
}

func (d *Driver) client() *api.Client {
	if d.Client == nil {
		d.Client = api.NewClient(d.Cred)
	}
	return d.Client
}

func (d *Driver) region() string {
	for _, r := range d.Regions {
		if r = strings.TrimSpace(r); r != "" && r != "all" {
			return r
		}
	}
	if r := strings.TrimSpace(d.Cred.Region); r != "" && r != "all" {
		return r
	}
	return defaultRegion
}

func (d *Driver) GetResource(ctx context.Context) (schema.Sms, error) {
	out := schema.Sms{}
	if d == nil {
		return out, errors.New("huawei msgsms: nil driver")
	}
	logger.Info("List Huawei MSGSMS signs and templates ...")
	region := d.region()

	var signs api.ListSmsSignResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "smsapi",
		Region:     region,
		Method:     http.MethodGet,
		Path:       "/v1/sms/signs",
		Idempotent: true,
	}, &signs); err != nil {
		return out, err
	}
	for _, s := range signs.Signs {
		out.Signs = append(out.Signs, schema.SmsSign{
			Name:   s.SignName,
			Type:   s.SignType,
			Status: firstNonEmpty(s.Status, s.Reason),
		})
	}

	var templates api.ListSmsTemplateResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "smsapi",
		Region:     region,
		Method:     http.MethodGet,
		Path:       "/v1/sms/templates",
		Idempotent: true,
	}, &templates); err != nil {
		return out, err
	}
	for _, t := range templates.Templates {
		out.Templates = append(out.Templates, schema.SmsTemplate{
			Name:    t.TemplateName,
			Status:  firstNonEmpty(t.Status, t.Reason),
			Content: t.Content,
		})
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
