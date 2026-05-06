package logging

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	defaultLogsPageSize = 100
	maxLogsPages        = 50
)

// GetLogs lists Cloud Logging log names per project. Each unique log name
// (e.g. `projects/<p>/logs/cloudaudit.googleapis.com%2Factivity`) is surfaced
// as one cloudlist `log` asset row.
func (d *Driver) GetLogs(ctx context.Context) ([]schema.Log, error) {
	out := []schema.Log{}
	if d == nil || d.Client == nil {
		return out, errors.New("gcp logging: nil api client")
	}
	logger.Info("List Cloud Logging log names ...")
	for _, project := range d.Projects {
		project = strings.TrimSpace(project)
		if project == "" {
			continue
		}
		names, err := d.listLogNames(ctx, project)
		if err != nil {
			return out, err
		}
		for _, fullName := range names {
			out = append(out, schema.Log{
				ProjectName: shortLogName(fullName),
				// GCP Cloud Logging is a global service; surface the project
				// in the Region column to make per-project distinctions
				// visible without changing the schema.
				Region:      project,
				Description: fullName,
			})
		}
	}
	return out, nil
}

func (d *Driver) listLogNames(ctx context.Context, project string) ([]string, error) {
	out := []string{}
	pageToken := ""
	for page := 0; page < maxLogsPages; page++ {
		query := url.Values{}
		query.Set("pageSize", strconv.Itoa(defaultLogsPageSize))
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		var resp api.ListLogsResponse
		if err := d.Client.Do(ctx, api.Request{
			Method:     http.MethodGet,
			BaseURL:    api.LoggingBaseURL,
			Path:       "/v2/projects/" + url.PathEscape(project) + "/logs",
			Query:      query,
			Idempotent: true,
		}, &resp); err != nil {
			return out, err
		}
		out = append(out, resp.LogNames...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}

// shortLogName trims the `projects/<p>/logs/` prefix and URL-decodes the
// remaining segment so the surfaced name is human-readable.
func shortLogName(full string) string {
	full = strings.TrimSpace(full)
	if idx := strings.LastIndex(full, "/logs/"); idx >= 0 {
		full = full[idx+len("/logs/"):]
	}
	if decoded, err := url.QueryUnescape(full); err == nil {
		return decoded
	}
	return full
}
