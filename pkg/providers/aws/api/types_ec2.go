package api

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const ec2APIVersion = "2016-11-15"

type EC2Region struct {
	Name string
}

type DescribeRegionsOutput struct {
	Regions []EC2Region
}

type EC2Tag struct {
	Key   string
	Value string
}

type EC2Instance struct {
	InstanceID    string
	PublicIP      string
	PrivateIP     string
	PublicDNSName string
	State         string
	Tags          []EC2Tag
}

type DescribeInstancesOutput struct {
	Instances []EC2Instance
	NextToken string
}

type describeRegionsResponse struct {
	XMLName    xml.Name            `xml:"DescribeRegionsResponse"`
	RegionInfo []ec2RegionInfoWire `xml:"regionInfo>item"`
}

type ec2RegionInfoWire struct {
	RegionName string `xml:"regionName"`
}

type describeInstancesResponse struct {
	XMLName      xml.Name             `xml:"DescribeInstancesResponse"`
	Reservations []ec2ReservationWire `xml:"reservationSet>item"`
	NextToken    string               `xml:"nextToken"`
}

type ec2ReservationWire struct {
	Instances []ec2InstanceWire `xml:"instancesSet>item"`
}

type ec2InstanceWire struct {
	InstanceID    string       `xml:"instanceId"`
	PublicIP      string       `xml:"ipAddress"`
	PrivateIP     string       `xml:"privateIpAddress"`
	PublicDNSName string       `xml:"dnsName"`
	State         ec2StateWire `xml:"instanceState"`
	Tags          []ec2TagWire `xml:"tagSet>item"`
}

type ec2StateWire struct {
	Name string `xml:"name"`
}

type ec2TagWire struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

func (c *Client) DescribeRegions(ctx context.Context, region string) (DescribeRegionsOutput, error) {
	var wire describeRegionsResponse
	err := c.DoXML(ctx, Request{
		Service:    "ec2",
		Region:     region,
		Action:     "DescribeRegions",
		Version:    ec2APIVersion,
		Method:     http.MethodPost,
		Path:       "/",
		Idempotent: true,
	}, &wire)
	if err != nil {
		return DescribeRegionsOutput{}, err
	}
	out := DescribeRegionsOutput{
		Regions: make([]EC2Region, 0, len(wire.RegionInfo)),
	}
	for _, region := range wire.RegionInfo {
		name := strings.TrimSpace(region.RegionName)
		if name == "" {
			continue
		}
		out.Regions = append(out.Regions, EC2Region{Name: name})
	}
	return out, nil
}

func (c *Client) DescribeInstances(ctx context.Context, region, nextToken string, maxResults int) (DescribeInstancesOutput, error) {
	query := url.Values{}
	if nextToken = strings.TrimSpace(nextToken); nextToken != "" {
		query.Set("NextToken", nextToken)
	}
	if maxResults > 0 {
		query.Set("MaxResults", strconv.Itoa(maxResults))
	}
	var wire describeInstancesResponse
	err := c.DoXML(ctx, Request{
		Service:    "ec2",
		Region:     region,
		Action:     "DescribeInstances",
		Version:    ec2APIVersion,
		Method:     http.MethodPost,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &wire)
	if err != nil {
		return DescribeInstancesOutput{}, err
	}
	out := DescribeInstancesOutput{
		Instances: make([]EC2Instance, 0),
		NextToken: strings.TrimSpace(wire.NextToken),
	}
	for _, reservation := range wire.Reservations {
		for _, instance := range reservation.Instances {
			item := EC2Instance{
				InstanceID:    strings.TrimSpace(instance.InstanceID),
				PublicIP:      strings.TrimSpace(instance.PublicIP),
				PrivateIP:     strings.TrimSpace(instance.PrivateIP),
				PublicDNSName: strings.TrimSpace(instance.PublicDNSName),
				State:         strings.TrimSpace(instance.State.Name),
				Tags:          make([]EC2Tag, 0, len(instance.Tags)),
			}
			for _, tag := range instance.Tags {
				item.Tags = append(item.Tags, EC2Tag{
					Key:   strings.TrimSpace(tag.Key),
					Value: strings.TrimSpace(tag.Value),
				})
			}
			out.Instances = append(out.Instances, item)
		}
	}
	return out, nil
}
