package plugins

import (
	"github.com/404tk/cloudtoolkit/pkg/providers/aws"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type AWS struct{}

func (p AWS) Check(block schema.OptionBlock) (schema.Provider, error) {
	return aws.New(block)
}

func (p AWS) Desc() string {
	return "Amazon Web Service"
}

func init() {
	registerProvider("aws", AWS{})
}
