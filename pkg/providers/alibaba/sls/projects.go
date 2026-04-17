package sls

import (
	"context"
	"strconv"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
)

type Driver struct {
	Cred   *credentials.StsTokenCredential
	Region string
}

func (d *Driver) newClient(region string) *Client {
	return NewClient(
		false, region, d.Cred.AccessKeyId, d.Cred.AccessKeySecret, d.Cred.AccessKeyStsToken)
}

// Reference https://api.aliyun.com/product/Sls
// Ignore by default，"ap-northeast-1", "ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-5", "ap-southeast-6", "ap-southeast-7", "us-east-1", "us-west-1", "eu-west-1", "eu-central-1", "ap-south-1", "me-east-1", "me-central-1", "cn-hangzhou-finance", "cn-shanghai-finance-1", "cn-shenzhen-finance-1", "cn-beijing-finance-1"
var all_regions = []string{"cn-qingdao", "cn-beijing", "cn-zhangjiakou", "cn-huhehaote", "cn-wulanchabu", "cn-hangzhou", "cn-shanghai", "cn-nanjing", "cn-shenzhen", "cn-heyuan", "cn-guangzhou", "cn-chengdu", "cn-hongkong"}

func (d *Driver) ListProjects(ctx context.Context) ([]schema.Log, error) {
	list := []schema.Log{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List SLS project ...")
	}
	var regions []string
	if d.Region == "all" {
		regions = all_regions
	} else {
		regions = append(regions, d.Region)
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Log, error) {
		var regionList []schema.Log
		client := d.newClient(r)
		var offset int32 = 0
		for {
			req := ListProjectRequest{Offset: offset, Size: 500}
			resp, err := client.ListProjects(req)
			if err != nil {
				return regionList, err
			}
			for _, project := range resp.Projects {
				_log := schema.Log{
					ProjectName: *project.ProjectName,
					Region:      *project.Region,
					Description: *project.Description,
				}
				timestamp, _ := strconv.ParseInt(*project.LastModifyTime, 10, 64)
				_log.LastModifyTime = time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
				regionList = append(regionList, _log)
			}
			if len(resp.Projects) < 500 {
				break
			}
			offset += 500
			select {
			case <-ctx.Done():
				return regionList, nil
			default:
			}
		}
		return regionList, nil
	})
	list = append(list, got...)
	return list, nil
}
