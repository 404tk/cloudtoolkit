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
	Resources(ctx context.Context) (Resources, error)
	UserManagement(action, username, password string)
	BucketDump(ctx context.Context, action, bucketName string)
	EventDump(action, args string)
	ExecuteCloudVMCommand(instanceID, cmd string)
	DBManagement(action, instanceID string)
}

// NewResources creates a new resources structure
func NewResources() Resources {
	return Resources{}
}

type Resources struct {
	Provider  string
	Hosts     []Host
	Storages  []Storage
	Users     []User
	Databases []Database
	Domains   []Domain
	Sms       Sms
	Logs      []Log
}

type Host struct {
	HostName    string `table:"HostName"`
	ID          string `table:"Instance ID"`
	State       string `table:"State"`
	PublicIPv4  string `table:"Public IP"`
	PrivateIpv4 string `table:"Private IP"`
	OSType      string `table:"OS Type"`
	DNSName     string `table:"DNS Name"`
	Public      bool   `table:"Public"`
	Region      string `table:"Region"`
}

type Storage struct {
	BucketName  string `table:"Bucket"`
	AccountName string `table:"Storage Account"`
	Region      string `table:"Region"`
}

type User struct {
	UserName    string `table:"User"`
	UserId      string `table:"ID"`
	Policies    string `table:"Policies"`
	EnableLogin bool   `table:"EnableLogin"`
	LastLogin   string `table:"LastLogin"`
	CreateTime  string `table:"CreateTime"`
}

type Database struct {
	InstanceId    string `table:"ID"`
	Engine        string `table:"Engine"`
	EngineVersion string `table:"Version"`
	Region        string `table:"Region"`
	Address       string `table:"Address"`
	NetworkType   string `table:"NetworkType"`
	DBNames       string `table:"DBName"`
}

type Domain struct {
	DomainName string
	Records    []Record
}

type Record struct {
	RR     string
	Type   string
	Value  string
	Status string
}

type Sms struct {
	Signs     []SmsSign
	Templates []SmsTemplate
	DailySize int64
}

type SmsSign struct {
	Name   string `table:"Name"`
	Type   string `table:"Type"`
	Status string `table:"Status"`
}

type SmsTemplate struct {
	Name    string `table:"Name"`
	Status  string `table:"Status"`
	Content string `table:"Content"`
}

type Event struct {
	Id        string
	Name      string
	Affected  string
	API       string
	Status    string
	SourceIp  string `table:"Source IP"`
	AccessKey string
	Time      string
}

type Log struct {
	ProjectName    string `table:"Project Name"`
	Region         string
	Description    string
	LastModifyTime string
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
type Options map[string]string

// GetMetadata returns the value for a key if it exists.
func (o Options) GetMetadata(key string) (string, bool) {
	data, ok := o[key]
	if !ok || data == "" {
		return "", false
	}
	return data, true
}
