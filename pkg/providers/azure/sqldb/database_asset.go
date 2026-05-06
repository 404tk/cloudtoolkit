package sqldb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

// GetDatabases lists Azure SQL servers across the visible subscriptions and
// surfaces them as the cloudlist `database` asset. SQL Database resources
// are nested under each server; we surface one row per server with the
// fully-qualified domain name as Address — that matches CSPM signal value
// (one server = one externally-reachable endpoint).
func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	out := []schema.Database{}
	if d == nil || d.Client == nil {
		return out, errors.New("azure sqldb: nil api client")
	}
	logger.Info("List Azure SQL servers ...")
	for _, sub := range d.SubscriptionIDs {
		servers, err := d.listServers(ctx, sub)
		if err != nil {
			logger.Error(fmt.Sprintf("List Azure SQL servers in %s: %s", sub, err.Error()))
			return out, err
		}
		for _, s := range servers {
			out = append(out, schema.Database{
				InstanceId:    s.Name,
				Engine:        "Microsoft.Sql",
				EngineVersion: s.Properties.Version,
				Region:        s.Location,
				Address:       s.Properties.FullyQualifiedDomainName,
				NetworkType:   networkType(s),
				DBNames:       "",
			})
		}
	}
	return out, nil
}

func (d *Driver) listServers(ctx context.Context, subscription string) ([]azapi.SQLServer, error) {
	pager := azapi.NewPager[azapi.SQLServer](d.Client, azapi.Request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Sql/servers", subscription),
		Query: url.Values{
			"api-version": {azapi.SQLAPIVersion},
		},
		Idempotent: true,
	})
	return pager.All(ctx)
}

// networkType returns "Public" if the server has a publicly-resolved FQDN
// (Azure SQL servers are publicly endpoint-routed by default unless an
// ARM private endpoint hides them).
func networkType(s azapi.SQLServer) string {
	if s.Properties.FullyQualifiedDomainName == "" {
		return "Private"
	}
	return "Public"
}
