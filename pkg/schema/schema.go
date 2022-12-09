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
	UserManagement(action, uname, pwd string)
}

// NewResources creates a new resources structure
func NewResources() *Resources {
	return &Resources{
		Hosts:     make([]*Host, 0),
		Storages:  make([]*Storage, 0),
		Users:     make([]*User, 0),
		Databases: make([]*Database, 0),
	}
}

type Resources struct {
	Provider  string
	Hosts     []*Host
	Storages  []*Storage
	Users     []*User
	Databases []*Database
}

type Host struct {
	PublicIPv4  string `table:"Public IP"`
	PrivateIpv4 string `table:"Private IP"`
	DNSName     string `table:"DNS Name"`
	Public      bool   `table:"Public"`
	Region      string `table:"Region"`
}

type Storage struct {
	BucketName     string `table:"Bucket"`
	FileSystemName string `table:"File System"`
	AccountName    string `table:"Account"`
	Region         string `table:"Region"`
}

type User struct {
	UserName    string `table:"User"`
	UserId      string `table:"ID"`
	EnableLogin bool   `table:"Enable Login"`
	LastLogin   string `table:"Last Login"`
	CreateTime  string `table:"Creat Time"`
}

type Database struct {
	DBInstanceId  string `table:"ID"`
	Engine        string `table:"Engine"`
	EngineVersion string `table:"Version"`
	Region        string `table:"Region"`
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
