// Package msgsms wraps Huawei Cloud MSGSMS template + sign listing for the
// cloudlist `sms` asset.
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

var supportedRegions = map[string]struct{}{
	"cn-north-4": {},
	"cn-south-1": {},
}

type Driver struct {
	Cred      auth.Credential
	Regions   []string
	DomainID  string
	Client    *api.Client
	projectID map[string]string

	ProjectCatalog *api.ProjectCatalog
}

func (d *Driver) client() *api.Client {
	if d.Client == nil {
		d.Client = api.NewClient(d.Cred)
	}
	return d.Client
}

func (d *Driver) region() (string, bool) {
	for _, r := range d.Regions {
		if r = strings.TrimSpace(r); isSupportedRegion(r) {
			return r, true
		}
	}
	if d.ProjectCatalog != nil {
		return "", false
	}
	if r := strings.TrimSpace(d.Cred.Region); isSupportedRegion(r) {
		return r, true
	}
	return defaultRegion, true
}

func (d *Driver) GetResource(ctx context.Context) (schema.Sms, error) {
	out := schema.Sms{}
	if d == nil {
		return out, errors.New("huawei msgsms: nil driver")
	}
	logger.Info("List Huawei MSGSMS signs and templates ...")
	region, ok := d.region()
	if !ok {
		return out, nil
	}
	projectID, err := d.resolveProjectID(ctx, region)
	if err != nil {
		return out, err
	}

	var signs api.ListSmsSignResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "msgsms",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v2/" + projectID + "/msgsms/signatures",
		Idempotent: true,
	}, &signs); err != nil {
		return out, err
	}
	for _, s := range signs.Results {
		out.Signs = append(out.Signs, schema.SmsSign{
			Name:   firstNonEmpty(s.SignName, s.SignID, s.ID),
			Type:   s.SignType,
			Status: firstNonEmpty(s.Status, s.Reason),
		})
	}

	var templates api.ListSmsTemplateResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "msgsms",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v2/" + projectID + "/msgsms/templates",
		Idempotent: true,
	}, &templates); err != nil {
		return out, err
	}
	for _, t := range templates.Results {
		out.Templates = append(out.Templates, schema.SmsTemplate{
			Name:    firstNonEmpty(t.TemplateName, t.TemplateID, t.ID),
			Status:  firstNonEmpty(t.Status, t.FlowStatus, t.Reason),
			Content: t.Content,
		})
	}
	return out, nil
}

func (d *Driver) resolveProjectID(ctx context.Context, region string) (string, error) {
	if projectID, ok := d.ProjectCatalog.ProjectID(region); ok {
		return projectID, nil
	}
	if d.ProjectCatalog != nil {
		return "", &api.ProjectNotFoundError{Region: region}
	}
	if d.projectID == nil {
		d.projectID = make(map[string]string)
	}
	if cached := strings.TrimSpace(d.projectID[region]); cached != "" {
		return cached, nil
	}
	pid, err := api.ResolveProjectID(ctx, d.client(), d.DomainID, region)
	if err != nil {
		return "", err
	}
	d.projectID[region] = pid
	return pid, nil
}

func isSupportedRegion(region string) bool {
	_, ok := supportedRegions[region]
	return ok
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
