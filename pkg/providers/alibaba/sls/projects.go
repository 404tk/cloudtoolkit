package sls

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
)

type Driver struct {
	Cred   *credentials.StsTokenCredential
	Region string
}

func (d *Driver) NewClient() *Client {
	return NewClient(
		false, d.Region, d.Cred.AccessKeyId, d.Cred.AccessKeySecret, d.Cred.AccessKeyStsToken)
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

	flag := false
	prevLength := 0
	count := 0
	for _, r := range regions {
		d.Region = r
		client := d.NewClient()
		var offset int32 = 0
		for {
			req := ListProjectRequest{
				Offset: offset,
				Size:   500,
			}
			resp, err := client.ListProject(req)
			if err != nil {
				logger.Error(err)
				goto done
			}
			for _, project := range resp.Projects {
				_log := schema.Log{
					ProjectName: *project.ProjectName,
					Region:      *project.Region,
					Description: *project.Description,
				}
				timestamp, _ := strconv.ParseInt(*project.LastModifyTime, 10, 64)
				_log.LastModifyTime = time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
				list = append(list, _log)
			}
			if len(resp.Projects) < 500 {
				break
			}
			offset += 500
		}
		select {
		case <-ctx.Done():
			goto done
		default:
			prevLength, flag = processbar.RegionPrint(r, len(list)-count, prevLength, flag)
			count = len(list)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
	}
	return list, nil
}
