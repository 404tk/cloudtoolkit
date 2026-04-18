package ec2

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type Driver struct {
	Config awsv2.Config
	Region string
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List EC2 instances ...")
	regions, err := d.GetEC2Regions(ctx)
	if err != nil {
		logger.Error("GetEC2Regions failed.")
		return list, err
	}
	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, region string) ([]schema.Host, error) {
		cfg := d.Config.Copy()
		if region != "" {
			cfg.Region = region
		}
		ec2Client := ec2.NewFromConfig(cfg)
		paginator := ec2.NewDescribeInstancesPaginator(ec2Client, &ec2.DescribeInstancesInput{
			MaxResults: awsv2.Int32(1000),
		})
		var items []schema.Host
		for paginator.HasMorePages() {
			resp, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, reservation := range resp.Reservations {
				for _, instance := range reservation.Instances {
					ip4 := awsv2.ToString(instance.PublicIpAddress)
					state := ""
					if instance.State != nil {
						state = string(instance.State.Name)
					}
					host := schema.Host{
						HostName:    "",
						ID:          awsv2.ToString(instance.InstanceId),
						State:       state,
						PublicIPv4:  ip4,
						PrivateIpv4: awsv2.ToString(instance.PrivateIpAddress),
						DNSName:     awsv2.ToString(instance.PublicDnsName),
						Public:      ip4 != "",
						Region:      region,
					}
					for _, tag := range instance.Tags {
						key := awsv2.ToString(tag.Key)
						if key == "aws:cloudformation:stack-name" || key == "Name" {
							host.HostName = awsv2.ToString(tag.Value)
							break
						}
					}
					items = append(items, host)
				}
			}
		}
		return items, nil
	})
	list = append(list, got...)
	return list, nil
}

func (d *Driver) GetEC2Regions(ctx context.Context) ([]string, error) {
	var regions []string
	if d.Region == "all" {
		ec2Client := ec2.NewFromConfig(d.Config)
		resp, err := ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
		if err != nil {
			return nil, err
		}
		for _, region := range resp.Regions {
			if name := awsv2.ToString(region.RegionName); name != "" {
				regions = append(regions, name)
			}
		}
	} else {
		region := d.Region
		if region == "" {
			region = d.Config.Region
		}
		regions = append(regions, region)
	}
	return regions, nil
}
