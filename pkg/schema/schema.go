package schema

import (
	"context"
	"fmt"
)

// Provider is an interface implemented by any cloud service provider.
//
// It provides the bare minimum of methods to allow complete overview of user
// data.
type Provider interface {
	// Name returns the name of the provider
	Name() string
	// Resources returns the provider for an resource deployment source.
	Resources(ctx context.Context) (*Resources, error)
}

// NewResources creates a new resources structure
func NewResources() *Resources {
	return &Resources{
		Hosts:     make([]*Host, 0),
		Storages:  make([]*Storage, 0),
		IAMs:      make([]*IAM, 0),
		Databases: make([]*Database, 0),
	}
}

type Resources struct {
	Provider  string
	Hosts     []*Host
	Storages  []*Storage
	IAMs      []*IAM
	Databases []*Database
}

type Host struct {
	Public      bool
	PublicIPv4  string
	PrivateIpv4 string
	DNSName     string
}

type Storage struct {
	BucketName     string
	FileSystemName string
	AccountName    string
	Region         string
}

type IAM struct {
	User   string
	UserId string
}

type Database struct {
	DBInstanceId  string
	Engine        string
	EngineVersion string
	Region        string
}

// ErrNoSuchKey means no such key exists in metadata.
type ErrNoSuchKey struct {
	Name string
}

// Error returns the value of the metadata key
func (e *ErrNoSuchKey) Error() string {
	return fmt.Sprintf("no such key: %s", e.Name)
}

// Options contains configuration options for a provider
type Options []OptionBlock

// OptionBlock is a single option on which operation is possible
type OptionBlock map[string]string

// GetMetadata returns the value for a key if it exists.
func (o OptionBlock) GetMetadata(key string) (string, bool) {
	data, ok := o[key]
	if !ok || data == "" {
		return "", false
	}
	return data, true
}
