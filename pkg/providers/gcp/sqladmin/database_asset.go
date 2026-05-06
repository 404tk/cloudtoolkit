package sqladmin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	listInstancesMaxPages = 50
)

// GetDatabases lists Cloud SQL instances across the configured projects and
// surfaces them as the cloudlist `database` asset. CSPM-relevant fields:
// engine version, primary IP address, and whether IPv4 is publicly enabled.
func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	out := []schema.Database{}
	if d == nil || d.Client == nil {
		return out, errors.New("gcp sqladmin: nil api client")
	}
	logger.Info("List Cloud SQL instances ...")
	for _, project := range d.Projects {
		project = strings.TrimSpace(project)
		if project == "" {
			continue
		}
		instances, err := d.listInstances(ctx, project)
		if err != nil {
			return out, err
		}
		for _, inst := range instances {
			out = append(out, schema.Database{
				InstanceId:    inst.Name,
				Engine:        databaseEngine(inst.DatabaseVersion),
				EngineVersion: inst.DatabaseVersion,
				Region:        inst.Region,
				Address:       primaryAddress(inst),
				NetworkType:   networkType(inst),
				DBNames:       inst.ConnectionName,
			})
		}
	}
	return out, nil
}

func (d *Driver) listInstances(ctx context.Context, project string) ([]api.SQLInstance, error) {
	out := []api.SQLInstance{}
	pageToken := ""
	for page := 0; page < listInstancesMaxPages; page++ {
		query := url.Values{}
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		var resp api.SQLInstancesListResponse
		if err := d.Client.Do(ctx, api.Request{
			Method:     http.MethodGet,
			BaseURL:    api.SQLAdminBaseURL,
			Path:       "/sql/v1beta4/projects/" + url.PathEscape(project) + "/instances",
			Query:      query,
			Idempotent: true,
		}, &resp); err != nil {
			return out, err
		}
		out = append(out, resp.Items...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}

// databaseEngine extracts the engine name from a Cloud SQL DatabaseVersion
// (e.g. "MYSQL_8_0" → "mysql", "POSTGRES_14" → "postgres").
func databaseEngine(version string) string {
	version = strings.ToLower(strings.TrimSpace(version))
	if version == "" {
		return ""
	}
	if idx := strings.Index(version, "_"); idx >= 0 {
		return version[:idx]
	}
	return version
}

// primaryAddress prefers the PRIMARY (public) IP. Cloud SQL also exposes
// PRIVATE and OUTGOING addresses; PRIMARY is the one CSPM detectors track for
// public-exposure signal.
func primaryAddress(inst api.SQLInstance) string {
	for _, ip := range inst.IPAddresses {
		if strings.EqualFold(ip.Type, "PRIMARY") {
			return ip.IPAddress
		}
	}
	for _, ip := range inst.IPAddresses {
		if ip.IPAddress != "" {
			return ip.IPAddress
		}
	}
	return ""
}

// networkType returns "Public" when ipv4 is enabled (a PRIMARY public IP is
// reachable) or "Private" otherwise.
func networkType(inst api.SQLInstance) string {
	if inst.Settings.IPConfiguration.IPv4Enabled {
		return "Public"
	}
	return "Private"
}
