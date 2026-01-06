package ecs

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

type Driver struct {
	Conf   *volcengine.Config
	Region string
}

func (d *Driver) NewClient(region string) (*ecs.ECS, error) {
	if region == "all" || region == "" {
		region = "cn-beijing"
	}
	sess, err := session.NewSession(d.Conf.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return ecs.New(sess), nil
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List ECS instances ...")
	svc, err := d.NewClient(d.Region)
	if err != nil {
		return list, err
	}

	var regions []string
	if d.Region == "all" {
		input := &ecs.DescribeRegionsInput{MaxResults: volcengine.Int32(100)}
		resp, err := svc.DescribeRegions(input)
		if err != nil {
			logger.Error("List regions failed.")
			return list, err
		}
		for _, r := range resp.Regions {
			regions = append(regions, *r.RegionId)
		}
	} else {
		regions = append(regions, d.Region)
	}
	flag := false
	prevLength := 0
	count := 0
	for _, r := range regions {
		svc, err = d.NewClient(r)
		if err != nil {
			continue
		}
		token := volcengine.String("")
		for {
			request := &ecs.DescribeInstancesInput{
				MaxResults: volcengine.Int32(100),
				NextToken:  token,
			}
			resp, err := svc.DescribeInstances(request)
			if err != nil {
				return list, err
			}
			for _, i := range resp.Instances {
				// Getting Host Information
				ipv4 := volcengine.StringValue(i.EipAddress.IpAddress)
				var privateIPv4 string
				if len(i.NetworkInterfaces) > 0 {
					privateIPv4 = volcengine.StringValue(i.NetworkInterfaces[0].PrimaryIpAddress)
				}

				_host := schema.Host{
					HostName:    volcengine.StringValue(i.Hostname),
					ID:          volcengine.StringValue(i.InstanceId),
					State:       volcengine.StringValue(i.Status),
					PublicIPv4:  ipv4,
					PrivateIpv4: privateIPv4,
					OSType:      volcengine.StringValue(i.OsType), // Windows or Linux
					Public:      ipv4 != "",
					Region:      r,
				}
				list = append(list, _host)
			}
			if len(resp.Instances) < 100 ||
				volcengine.StringValue(resp.NextToken) == "" {
				break
			}
			token = resp.NextToken
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
