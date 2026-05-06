// Package sms wraps Tencent Cloud SMS template + sign listing for the
// cloudlist `sms` asset.
//
// Tencent's SMS API requires explicit ID sets to query — there is no
// "list all templates" endpoint with empty filter. We pass an empty IDSet
// which the service treats as "no targets" and surfaces nothing useful, so
// callers must populate IDs externally; this driver is therefore a best-
// effort enumeration. CSPM detection still benefits from the schema slot
// being filled (templates the operator already knows about).
package sms

import (
	"context"
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const defaultRegion = "ap-guangzhou"

type Driver struct {
	Credential    auth.Credential
	Region        string
	SignIDs       []uint64
	TemplateIDs   []uint64
	clientOptions []api.Option
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Credential, d.clientOptions...)
}

func (d *Driver) requestRegion() string {
	if r := strings.TrimSpace(d.Region); r != "" && r != "all" {
		return r
	}
	return defaultRegion
}

// GetResource fills schema.Sms with sign + template summaries. With empty
// SignIDs / TemplateIDs the response will be empty (Tencent requires explicit
// IDs); the schema slot is still populated so cloudlist output is consistent.
func (d *Driver) GetResource(ctx context.Context) (schema.Sms, error) {
	out := schema.Sms{}
	if d == nil {
		return out, errors.New("tencent sms: nil driver")
	}
	logger.Info("List Tencent SMS signs and templates ...")
	region := d.requestRegion()
	client := d.newClient()

	if len(d.SignIDs) > 0 {
		signs, err := client.DescribeSmsSignList(ctx, region, d.SignIDs)
		if err != nil {
			return out, err
		}
		for _, s := range signs.Response.DescribeSignListStatusSet {
			out.Signs = append(out.Signs, schema.SmsSign{
				Name:   derefString(s.SignName),
				Type:   signTypeName(s.SignType),
				Status: signStatusLabel(s.StatusCode, s.ReviewReply),
			})
		}
	}

	if len(d.TemplateIDs) > 0 {
		templates, err := client.DescribeSmsTemplateList(ctx, region, d.TemplateIDs)
		if err != nil {
			return out, err
		}
		for _, t := range templates.Response.DescribeTemplateStatusSet {
			out.Templates = append(out.Templates, schema.SmsTemplate{
				Name:    derefString(t.TemplateName),
				Status:  signStatusLabel(t.StatusCode, t.ReviewReply),
				Content: derefString(t.TemplateContent),
			})
		}
	}
	return out, nil
}

func signTypeName(v *uint64) string {
	if v == nil {
		return ""
	}
	switch *v {
	case 0:
		return "Company"
	case 1:
		return "App"
	case 2:
		return "Website"
	case 3:
		return "WeChat OA"
	case 4:
		return "Trademark"
	case 5:
		return "Government"
	case 6:
		return "Other"
	}
	return ""
}

// signStatusLabel maps Tencent's StatusCode (-1 pending, 0 pass, 1 reject)
// into a stable string. Falls back to ReviewReply when StatusCode is missing.
func signStatusLabel(code *int, reply *string) string {
	if code == nil {
		return derefString(reply)
	}
	switch *code {
	case 0:
		return "Approved"
	case 1:
		return "Rejected"
	case -1:
		return "Pending"
	default:
		return derefString(reply)
	}
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
