package iam

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Cred     auth.Credential
	Client   *api.Client
	Username string
	Password string
	DomainID string
}

func (d *Driver) client() *api.Client {
	if d.Client == nil {
		d.Client = api.NewClient(d.Cred)
	}
	return d.Client
}

func (d *Driver) requestRegion() (string, error) {
	region := strings.TrimSpace(d.Cred.Region)
	switch region {
	case "":
		return "", fmt.Errorf("huawei iam: empty region")
	case "all":
		return "", fmt.Errorf("huawei iam: unresolved region %q", region)
	default:
		return region, nil
	}
}

func (d *Driver) domainHeaders(ctx context.Context) http.Header {
	domainID := d.resolveDomainID(ctx)
	if domainID == "" {
		return nil
	}
	headers := http.Header{}
	headers.Set("X-Domain-Id", domainID)
	return headers
}

func (d *Driver) resolveDomainID(ctx context.Context) string {
	if strings.TrimSpace(d.DomainID) != "" {
		return d.DomainID
	}
	resp, err := d.listAuthDomains(ctx)
	if err != nil || len(resp.Domains) == 0 {
		return ""
	}
	if len(resp.Domains) > 1 {
		logger.Warning(fmt.Sprintf("Multiple domains visible (%d); proceeding with first %q. Verify this matches your AK's domain.", len(resp.Domains), resp.Domains[0].ID))
	}
	d.DomainID = resp.Domains[0].ID
	return d.DomainID
}

func (d *Driver) listAuthDomains(ctx context.Context) (api.ListAuthDomainsResponse, error) {
	var resp api.ListAuthDomainsResponse
	region, err := d.requestRegion()
	if err != nil {
		return resp, err
	}
	err = d.client().DoJSON(ctx, api.Request{
		Service:    "iam",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v3/auth/domains",
		Idempotent: true,
	}, &resp)
	return resp, err
}
