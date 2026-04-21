package sls

import (
	"context"
	"net/http"
	"strconv"
	"time"

	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

type Driver struct {
	Cred       aliauth.Credential
	Region     string
	httpClient *http.Client
	partialErr error
}

func (d *Driver) newClient(region string) *Client {
	client := NewClient(false, region, d.Cred.AccessKeyID, d.Cred.AccessKeySecret, d.Cred.SecurityToken)
	if d.httpClient != nil {
		client.httpClient = d.httpClient
	}
	return client
}

// Reference https://api.aliyun.com/product/Sls
// Ignore by default，"ap-northeast-1", "ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-5", "ap-southeast-6", "ap-southeast-7", "us-east-1", "us-west-1", "eu-west-1", "eu-central-1", "ap-south-1", "me-east-1", "me-central-1", "cn-hangzhou-finance", "cn-shanghai-finance-1", "cn-shenzhen-finance-1", "cn-beijing-finance-1"
var allRegions = []string{"cn-qingdao", "cn-beijing", "cn-zhangjiakou", "cn-huhehaote", "cn-wulanchabu", "cn-hangzhou", "cn-shanghai", "cn-nanjing", "cn-shenzhen", "cn-heyuan", "cn-guangzhou", "cn-chengdu", "cn-hongkong"}

func (d *Driver) ListProjects(ctx context.Context) ([]schema.Log, error) {
	list := []schema.Log{}
	d.partialErr = nil
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List SLS project ...")
	}
	var regions []string
	if d.Region == "all" {
		regions = allRegions
	} else {
		regions = append(regions, d.Region)
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Log, error) {
		client := d.newClient(r)
		return paginate.Fetch(ctx, func(ctx context.Context, offset int32) (paginate.Page[schema.Log, int32], error) {
			resp, err := client.ListProjects(ListProjectRequest{Offset: offset, Size: 500})
			if err != nil {
				return paginate.Page[schema.Log, int32]{}, err
			}
			items := make([]schema.Log, 0, len(resp.Projects))
			for _, project := range resp.Projects {
				timestamp, _ := strconv.ParseInt(*project.LastModifyTime, 10, 64)
				items = append(items, schema.Log{
					ProjectName:    *project.ProjectName,
					Region:         *project.Region,
					Description:    *project.Description,
					LastModifyTime: time.Unix(timestamp, 0).Format("2006-01-02 15:04:05"),
				})
			}
			return paginate.Page[schema.Log, int32]{
				Items: items,
				Next:  offset + 500,
				Done:  len(resp.Projects) < 500,
			}, nil
		})
	})
	list = append(list, got...)
	d.partialErr = regionrun.Wrap(regionErrs)
	return list, nil
}

func (d *Driver) PartialError() error {
	return d.partialErr
}
