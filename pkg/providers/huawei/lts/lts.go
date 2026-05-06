// Package lts wraps the Huawei Cloud LTS (Log Tank Service) ListLogGroups
// action for the cloudlist `log` asset.
package lts

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const defaultRegion = "cn-north-4"

// Driver enumerates LTS log groups inside the resolved project.
type Driver struct {
	Cred      auth.Credential
	Regions   []string
	DomainID  string
	Client    *api.Client
	projectID map[string]string
}

func (d *Driver) client() *api.Client {
	if d.Client == nil {
		d.Client = api.NewClient(d.Cred)
	}
	return d.Client
}

// GetLogs lists LTS log groups in the resolved region. LTS is per-region but
// the validation flow only iterates the primary region for now (consistent
// with the CTS event-check driver).
func (d *Driver) GetLogs(ctx context.Context) ([]schema.Log, error) {
	out := []schema.Log{}
	if d == nil {
		return out, errors.New("huawei lts: nil driver")
	}
	logger.Info("List Huawei LTS log groups ...")
	region := d.region()
	projectID, err := d.resolveProjectID(ctx, region)
	if err != nil {
		return out, err
	}
	var resp api.ListLogGroupsResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "lts",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v2/" + projectID + "/groups",
		Idempotent: true,
	}, &resp); err != nil {
		return out, err
	}
	for _, g := range resp.LogGroups {
		out = append(out, schema.Log{
			ProjectName:    g.LogGroupName,
			Region:         region,
			Description:    g.LogGroupID,
			LastModifyTime: formatLTSTime(g.CreationTime),
		})
	}
	return out, nil
}

func (d *Driver) resolveProjectID(ctx context.Context, region string) (string, error) {
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

// formatLTSTime converts the LTS millisecond epoch into the human readable
// format `YYYY-MM-DD HH:MM:SS`.
func formatLTSTime(epochMillis int64) string {
	if epochMillis <= 0 {
		return ""
	}
	return time.Unix(epochMillis/1000, 0).UTC().Format("2006-01-02 15:04:05")
}
