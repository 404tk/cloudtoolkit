package loganalytics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

// Driver enumerates Log Analytics workspaces across the visible
// subscriptions and surfaces them as the cloudlist `log` asset. A workspace
// is the Azure container that aggregates logs from VMs, Application
// Insights, Defender for Cloud, etc.
type Driver struct {
	Client          *azapi.Client
	SubscriptionIDs []string
}

func (d *Driver) GetLogs(ctx context.Context) ([]schema.Log, error) {
	list := []schema.Log{}
	if d == nil || d.Client == nil {
		return list, errors.New("azure log analytics: nil api client")
	}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List Azure Log Analytics workspaces ...")
	}

	for _, sub := range d.SubscriptionIDs {
		ws, err := d.listWorkspaces(ctx, sub)
		if err != nil {
			logger.Error(fmt.Sprintf("List Log Analytics workspaces in %s: %s", sub, err.Error()))
			return list, err
		}
		for _, w := range ws {
			list = append(list, schema.Log{
				ProjectName:    strings.TrimSpace(w.Name),
				Region:         strings.TrimSpace(w.Location),
				Description:    workspaceDescription(w),
				LastModifyTime: strings.TrimSpace(w.Properties.ModifiedDate),
			})
		}
	}
	return list, nil
}

func (d *Driver) listWorkspaces(ctx context.Context, subscription string) ([]azapi.Workspace, error) {
	pager := azapi.NewPager[azapi.Workspace](d.Client, azapi.Request{
		Method:     http.MethodGet,
		Path:       fmt.Sprintf("/subscriptions/%s/providers/Microsoft.OperationalInsights/workspaces", subscription),
		Query:      url.Values{"api-version": {azapi.OperationalInsightsAPIVersion}},
		Idempotent: true,
	})
	return pager.All(ctx)
}

func workspaceDescription(w azapi.Workspace) string {
	parts := []string{}
	if w.Properties.CustomerID != "" {
		parts = append(parts, "customer="+w.Properties.CustomerID)
	}
	if w.Properties.RetentionInDays > 0 {
		parts = append(parts, fmt.Sprintf("retention=%d days", w.Properties.RetentionInDays))
	}
	if w.Properties.Sku != nil && w.Properties.Sku.Name != "" {
		parts = append(parts, "sku="+w.Properties.Sku.Name)
	}
	return strings.Join(parts, "; ")
}
