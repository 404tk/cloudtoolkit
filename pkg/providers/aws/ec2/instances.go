package ec2

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Driver struct {
	Session *session.Session
	Region  string
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List EC2 instances ...")
	regions, err := d.GetEC2Regions()
	if err != nil {
		logger.Error("GetEC2Regions failed.")
		return list, err
	}
	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, region string) ([]schema.Host, error) {
		ec2Client := ec2.New(d.Session, aws.NewConfig().WithRegion(region))
		return paginate.Fetch(ctx, func(ctx context.Context, token *string) (paginate.Page[schema.Host, *string], error) {
			req := &ec2.DescribeInstancesInput{MaxResults: aws.Int64(1000), NextToken: token}
			resp, err := ec2Client.DescribeInstancesWithContext(ctx, req)
			if err != nil {
				return paginate.Page[schema.Host, *string]{}, err
			}
			var items []schema.Host
			for _, reservation := range resp.Reservations {
				for _, instance := range reservation.Instances {
					ip4 := aws.StringValue(instance.PublicIpAddress)
					host := schema.Host{
						State:       instance.State.String(),
						PublicIPv4:  ip4,
						PrivateIpv4: aws.StringValue(instance.PrivateIpAddress),
						DNSName:     aws.StringValue(instance.PublicDnsName),
						Public:      ip4 != "",
						Region:      region,
					}
					for _, tag := range instance.Tags {
						if *tag.Key == "aws:cloudformation:stack-name" || *tag.Key == "Name" {
							host.HostName = *tag.Value
							break
						}
					}
					items = append(items, host)
				}
			}
			return paginate.Page[schema.Host, *string]{
				Items: items,
				Next:  resp.NextToken,
				Done:  aws.StringValue(resp.NextToken) == "",
			}, nil
		})
	})
	list = append(list, got...)
	return list, nil
}

func (d *Driver) GetEC2Regions() ([]string, error) {
	var regions []string
	ec2Client := ec2.New(d.Session)
	if d.Region == "all" {
		resp, err := ec2Client.DescribeRegions(&ec2.DescribeRegionsInput{})
		if err != nil {
			return nil, err
		}
		for _, region := range resp.Regions {
			regions = append(regions, aws.StringValue(region.RegionName))
		}
	} else {
		regions = append(regions, d.Region)
	}
	return regions, nil
}
