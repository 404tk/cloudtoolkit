package schema

import (
	"context"
	"fmt"
	"strings"
)

// Provider is the minimum contract every cloud must satisfy. Capability
// interfaces below (Enumerator, IAMManager, etc.) extend it optionally; a
// payload type-asserts for the capability it needs and fails gracefully
// when the current provider does not implement it.
type Provider interface {
	Name() string
}

// Enumerator powers the asset-inventory (`cloudlist`) payload.
type Enumerator interface {
	Provider
	Resources(ctx context.Context) (Resources, error)
}

// IAMManager powers the iam-user-check payload.
type IAMManager interface {
	Provider
	UserManagement(action, username, password string) (IAMResult, error)
}

type IAMResult struct {
	Action    string
	Username  string
	Password  string
	LoginURL  string
	AccountID string
	Message   string
}

// BucketManager powers the bucket-check payload.
type BucketManager interface {
	Provider
	BucketDump(ctx context.Context, action, bucketName string) ([]BucketResult, error)
}

type BucketResult struct {
	Action      string
	BucketName  string
	ObjectCount int64
	Objects     []BucketObject
	Message     string
}

type BucketObject struct {
	BucketName   string
	Key          string
	Size         int64
	LastModified string
	StorageClass string
}

func AggregateBucketResults(action, bucketName string, results []BucketResult) BucketResult {
	result := BucketResult{
		Action:     action,
		BucketName: bucketName,
	}
	if len(results) == 0 {
		result.Message = "no buckets found"
		return result
	}
	if len(results) == 1 {
		result = results[0]
		if result.Action == "" {
			result.Action = action
		}
		if result.BucketName == "" {
			result.BucketName = bucketName
		}
		return result
	}
	for _, item := range results {
		result.ObjectCount += item.ObjectCount
		result.Objects = append(result.Objects, item.Objects...)
	}
	if result.ObjectCount > 0 {
		result.Message = fmt.Sprintf("%d buckets, %d total objects", len(results), result.ObjectCount)
	} else {
		result.Message = fmt.Sprintf("%d buckets", len(results))
	}
	return result
}

// EventReader powers the event-check payload.
type EventReader interface {
	Provider
	EventDump(action, args string)
}

// VMExecutor powers the instance-cmd-check / shell payloads.
type VMExecutor interface {
	Provider
	ExecuteCloudVMCommand(instanceID, cmd string)
}

// DBManager powers the rds-account-check payload.
type DBManager interface {
	Provider
	DBManagement(action, instanceID string)
}

// Asset is any cloud resource that can be enumerated and rendered. New asset
// types (FaaS, K8s clusters, container registries, etc.) only need to
// implement AssetType() to flow through the existing asset-inventory pipeline.
type Asset interface {
	AssetType() string
}

// Asset type constants. Providers and payloads should reference these rather
// than raw strings to keep the grouping key canonical.
const (
	AssetHost     = "host"
	AssetStorage  = "storage"
	AssetUser     = "user"
	AssetDatabase = "database"
	AssetDomain   = "domain"
	AssetLog      = "log"
)

// NewResources creates a new resources structure
func NewResources() Resources {
	return Resources{}
}

type Resources struct {
	Provider string
	Assets   []Asset
	Sms      Sms
	Errors   []ResourceError
}

type ResourceError struct {
	Scope   string
	Message string
}

type resourceErrorExpander interface {
	ResourceErrors(scope string) []ResourceError
}

func (r *Resources) AddError(scope string, err error) {
	if err == nil {
		return
	}
	if expander, ok := err.(resourceErrorExpander); ok {
		r.Errors = append(r.Errors, expander.ResourceErrors(scope)...)
		return
	}
	r.Errors = append(r.Errors, ResourceError{
		Scope:   scope,
		Message: err.Error(),
	})
}

func (r Resources) Err() error {
	if len(r.Errors) == 0 {
		return nil
	}
	messages := make([]string, 0, len(r.Errors))
	for _, item := range r.Errors {
		if item.Scope == "" {
			messages = append(messages, item.Message)
			continue
		}
		messages = append(messages, fmt.Sprintf("%s: %s", item.Scope, item.Message))
	}
	return fmt.Errorf("partial enumeration errors: %s", strings.Join(messages, "; "))
}

// Grouped returns assets partitioned by AssetType() while preserving insertion
// order within each bucket. Used by the asset-inventory printer so each asset
// type renders as its own table.
func (r *Resources) Grouped() map[string][]Asset {
	out := make(map[string][]Asset)
	for _, a := range r.Assets {
		t := a.AssetType()
		out[t] = append(out[t], a)
	}
	return out
}

// AppendAssets copies a typed slice into r.Assets as Asset values. Provider
// implementations use this to flow a []Host / []Storage / ... into the open
// asset list without writing the boxing loop inline.
func AppendAssets[T Asset](r *Resources, items []T) {
	for _, i := range items {
		r.Assets = append(r.Assets, i)
	}
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

func (Host) AssetType() string { return AssetHost }

type Storage struct {
	BucketName  string `table:"Bucket"`
	AccountName string `table:"Storage Account"`
	Region      string `table:"Region"`
}

func (Storage) AssetType() string { return AssetStorage }

type User struct {
	UserName    string `table:"User"`
	UserId      string `table:"ID"`
	Policies    string `table:"Policies"`
	EnableLogin bool   `table:"EnableLogin"`
	LastLogin   string `table:"LastLogin"`
	CreateTime  string `table:"CreateTime"`
}

func (User) AssetType() string { return AssetUser }

type Database struct {
	InstanceId    string `table:"ID"`
	Engine        string `table:"Engine"`
	EngineVersion string `table:"Version"`
	Region        string `table:"Region"`
	Address       string `table:"Address"`
	NetworkType   string `table:"NetworkType"`
	DBNames       string `table:"DBName"`
}

func (Database) AssetType() string { return AssetDatabase }

type Domain struct {
	DomainName string
	Records    []Record
}

func (Domain) AssetType() string { return AssetDomain }

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

func (Log) AssetType() string { return AssetLog }

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
