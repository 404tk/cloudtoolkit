package ec2

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// InstanceProvider is an instance provider for aws API
type InstanceProvider struct {
	Session *session.Session
	Region  string
}

// GetResource returns all the resources in the store for a provider.
func (d *InstanceProvider) GetResource(ctx context.Context) ([]*schema.Host, error) {
	list := schema.NewResources().Hosts
	log.Println("[*] Start enumerating EC2 ...")
	count := 0
	regions, err := d.GetEC2Regions()
	if err != nil {
		log.Println("[-] Enumerate EC2 failed.")
		return list, err
	}
	for _, region := range regions {
		req := &ec2.DescribeInstancesInput{
			MaxResults: aws.Int64(1000),
		}
		endpointBuilder := &strings.Builder{}
		endpointBuilder.WriteString("https://ec2.")
		endpointBuilder.WriteString(region)
		endpointBuilder.WriteString(".amazonaws.com")

		ec2Client := ec2.New(
			d.Session,
			aws.NewConfig().WithEndpoint(endpointBuilder.String()),
			aws.NewConfig().WithRegion(region),
		)
		for {
			resp, err := ec2Client.DescribeInstances(req)
			if err != nil {
				return list, err
			}
			for _, reservation := range resp.Reservations {
				for _, instance := range reservation.Instances {
					ip4 := aws.StringValue(instance.PublicIpAddress)
					host := schema.Host{
						PublicIPv4:  ip4,
						PrivateIpv4: aws.StringValue(instance.PrivateIpAddress),
						DNSName:     aws.StringValue(instance.PublicDnsName),
						Public:      ip4 != "",
						Region:      region,
					}
					list = append(list, &host)
				}
			}
			if aws.StringValue(resp.NextToken) == "" {
				break
			}
			req.SetNextToken(aws.StringValue(resp.NextToken))
		}
		progress := fmt.Sprintf("Inquiring %s regionId,number of discovered hosts: %d", region, len(list)-count)
		log.Println(progress)
		count = len(list)
	}
	return list, nil
}

func (d *InstanceProvider) GetEC2Regions() ([]string, error) {
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
