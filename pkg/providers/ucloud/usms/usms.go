// Package usms wraps UCloud USMS for the cloudlist `sms` asset.
// Pattern-inferred — see api/types_usms.go.
package usms

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	defaultRegion = "cn-bj2"
	pageSize      = 100
	maxPages      = 50
)

type Driver struct {
	Credential ucloudauth.Credential
	Client     *api.Client
	ProjectID  string
	Region     string
}

func (d *Driver) client() *api.Client {
	if d.Client != nil {
		return d.Client
	}
	return api.NewClient(d.Credential)
}

func (d *Driver) requestRegion() string {
	if r := strings.TrimSpace(d.Region); r != "" && r != "all" {
		return r
	}
	return defaultRegion
}

func (d *Driver) GetResource(ctx context.Context) (schema.Sms, error) {
	out := schema.Sms{}
	if d == nil {
		return out, errors.New("ucloud usms: nil driver")
	}
	logger.Info("List UCloud USMS signatures and templates ...")
	region := d.requestRegion()
	client := d.client()

	offset := 0
	for page := 0; page < maxPages; page++ {
		params := map[string]any{"Region": region, "Limit": strconv.Itoa(pageSize), "Offset": strconv.Itoa(offset)}
		if d.ProjectID != "" {
			params["ProjectId"] = d.ProjectID
		}
		var resp api.DescribeUSMSSignatureResponse
		if err := client.Do(ctx, api.Request{Action: "DescribeUSMSSignature", Params: params, Idempotent: true}, &resp); err != nil {
			return out, err
		}
		for _, s := range resp.Signatures {
			out.Signs = append(out.Signs, schema.SmsSign{
				Name:   s.SigContent,
				Type:   sigTypeName(s.SigType),
				Status: usmsStatusLabel(s.Status, s.ErrMsg),
			})
		}
		if len(resp.Signatures) < pageSize {
			break
		}
		offset += len(resp.Signatures)
	}

	offset = 0
	for page := 0; page < maxPages; page++ {
		params := map[string]any{"Region": region, "Limit": strconv.Itoa(pageSize), "Offset": strconv.Itoa(offset)}
		if d.ProjectID != "" {
			params["ProjectId"] = d.ProjectID
		}
		var resp api.DescribeUSMSTemplateResponse
		if err := client.Do(ctx, api.Request{Action: "DescribeUSMSTemplate", Params: params, Idempotent: true}, &resp); err != nil {
			return out, err
		}
		for _, t := range resp.Templates {
			out.Templates = append(out.Templates, schema.SmsTemplate{
				Name:    firstNonEmpty(t.TemplateName, t.TemplateID),
				Status:  usmsStatusLabel(t.Status, t.ErrMsg),
				Content: t.Template,
			})
		}
		if len(resp.Templates) < pageSize {
			break
		}
		offset += len(resp.Templates)
	}

	return out, nil
}

func usmsStatusLabel(code int, errMsg string) string {
	switch code {
	case 0:
		return "Approved"
	case 1:
		return "Pending"
	case 2:
		return "Rejected"
	}
	if errMsg != "" {
		return errMsg
	}
	return ""
}

func sigTypeName(code int) string {
	switch code {
	case 0:
		return "Company"
	case 1:
		return "App"
	case 2:
		return "Website"
	case 3:
		return "Trademark"
	case 4:
		return "Government"
	case 5:
		return "Other"
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
