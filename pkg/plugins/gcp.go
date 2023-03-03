//go:build !no_gcp

package plugins

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type GCP struct{}

func (p GCP) Check(block schema.Options) (schema.Provider, error) {
	return gcp.New(block)
}

func (p GCP) Desc() string {
	return "Google Cloud Platform"
}

func init() {
	registerProvider("gcp", GCP{})
}
