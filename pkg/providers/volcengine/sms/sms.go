package sms

import (
	"context"
	"errors"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	pageSize = 100
	maxPages = 50
)

type Driver struct {
	Client *api.Client
	Region string
}

func (d *Driver) GetResource(ctx context.Context) (schema.Sms, error) {
	out := schema.Sms{}
	if d == nil || d.Client == nil {
		return out, errors.New("volcengine sms: nil api client")
	}
	logger.Info("List Volcengine SMS signs and templates ...")
	subAccounts, err := d.listSubAccounts(ctx)
	if err != nil {
		return out, err
	}
	if len(subAccounts) == 0 {
		return out, nil
	}

	for _, subAccount := range subAccounts {
		for page := 1; page <= maxPages; page++ {
			resp, err := d.Client.ListSmsSigns(ctx, subAccount, page, pageSize)
			if err != nil {
				return out, err
			}
			signs := resp.Signs()
			for _, s := range signs {
				out.Signs = append(out.Signs, schema.SmsSign{
					Name:   firstNonEmpty(s.Sign, s.Content, s.ID, s.SignID),
					Type:   firstNonEmpty(s.SignType, s.Source),
					Status: firstNonEmpty(s.Status, smsStatus(s.StatusCode), s.Reason, s.ReasonText),
				})
			}
			if len(signs) < pageSize {
				break
			}
		}

		for page := 1; page <= maxPages; page++ {
			resp, err := d.Client.ListSmsTemplates(ctx, subAccount, page, pageSize)
			if err != nil {
				return out, err
			}
			templates := resp.Templates()
			for _, t := range templates {
				out.Templates = append(out.Templates, schema.SmsTemplate{
					Name:    firstNonEmpty(t.TemplateName, t.Name, t.TemplateID, t.TemplateIDLower, t.ID),
					Status:  firstNonEmpty(t.Status, smsStatus(t.StatusCode), t.Reason, t.ReasonText),
					Content: firstNonEmpty(t.Content, t.Text),
				})
			}
			if len(templates) < pageSize {
				break
			}
		}
	}

	return out, nil
}

func (d *Driver) listSubAccounts(ctx context.Context) ([]string, error) {
	subAccounts := []string{}
	for page := 1; page <= maxPages; page++ {
		resp, err := d.Client.ListSmsSubAccounts(ctx, page, pageSize)
		if err != nil {
			return nil, err
		}
		for _, account := range resp.Result.List {
			if name := firstNonEmpty(account.SubAccountID, account.SubAccount, account.SubAccountName); name != "" {
				subAccounts = append(subAccounts, name)
			}
		}
		if len(resp.Result.List) < pageSize {
			break
		}
	}
	return subAccounts, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func smsStatus(code int64) string {
	switch code {
	case 1:
		return "reviewing"
	case 2:
		return "rejected"
	case 3:
		return "passed"
	case 4:
		return "closed"
	case 5:
		return "exempted"
	default:
		return ""
	}
}
