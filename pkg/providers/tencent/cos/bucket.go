package cos

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Credential    auth.Credential
	Client        *Client
	clientOptions []Option
}

// NewDriver creates a COS driver with optional client injections.
func NewDriver(cred auth.Credential, opts ...Option) *Driver {
	return &Driver{
		Credential:    cred,
		clientOptions: append([]Option(nil), opts...),
	}
}

// SetClientOptions replaces the client options used by lazy client creation.
func (d *Driver) SetClientOptions(opts ...Option) {
	d.clientOptions = append([]Option(nil), opts...)
}

func (d *Driver) client() *Client {
	if d.Client == nil {
		d.Client = NewClient(d.Credential, d.clientOptions...)
	}
	return d.Client
}

func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List COS buckets ...")
	}
	resp, err := d.client().ListBuckets(ctx)
	if err != nil {
		logger.Error("List buckets failed.")
		return nil, err
	}

	for _, bucket := range resp.Buckets {
		_bucket := schema.Storage{
			BucketName: bucket.Name,
			Region:     bucket.Region,
		}

		list = append(list, _bucket)
	}

	return list, nil
}
