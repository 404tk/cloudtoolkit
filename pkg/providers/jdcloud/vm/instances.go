package vm

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/jdcloud-api/jdcloud-sdk-go/core"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/vm/client"
)

type Driver struct {
	Cred   *core.Credential
	Token  string
	Region string
}

func (d *Driver) newClient() *client.VmClient {
	c := client.NewVmClient(d.Cred)
	c.DisableLogger()
	return c
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List VM instances ...")

	region := d.Region
	if d.Region == "all" {
		region = "cn-north-1"
	}
	svc := d.newClient()
	req := apis.NewDescribeInstancesRequest(region)
	req.AddHeader("x-jdcloud-security-token", d.Token)
	resp, err := svc.DescribeInstances(req)
	if err != nil {
		logger.Error("List instances failed.")
		return list, err
	}

	for _, i := range resp.Result.Instances {
		ipv4 := i.ElasticIpAddress
		_host := schema.Host{
			HostName:    i.Hostname,
			ID:          i.InstanceId,
			State:       i.Status,
			PublicIPv4:  ipv4,
			PrivateIpv4: i.PrivateIpAddress,
			OSType:      i.OsType, // windows or linux
			Public:      ipv4 != "",
			Region:      region,
		}
		list = append(list, _host)
	}

	return list, nil
}
