package ecs

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
)

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Cred, d.clientOptions...)
}
