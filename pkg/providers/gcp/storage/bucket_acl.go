package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const (
	allUsersMember         = "allUsers"
	objectViewerRole       = "roles/storage.objectViewer"
	legacyBucketReaderRole = "roles/storage.legacyBucketReader"
)

// AuditBucketACL returns the public-access posture of each bucket.
func (d *Driver) AuditBucketACL(ctx context.Context, container string) ([]schema.BucketACLEntry, error) {
	if d == nil || d.Client == nil {
		return nil, errors.New("gcp storage: nil api client")
	}
	buckets := []string{}
	if strings.TrimSpace(container) != "" {
		buckets = append(buckets, container)
	} else {
		all, err := d.GetBuckets(ctx)
		if err != nil {
			return nil, err
		}
		for _, b := range all {
			buckets = append(buckets, b.BucketName)
		}
	}
	out := make([]schema.BucketACLEntry, 0, len(buckets))
	for _, name := range buckets {
		policy, err := d.getBucketIamPolicy(ctx, name)
		if err != nil {
			return out, err
		}
		level := "Private"
		for _, b := range policy.Bindings {
			if isPublicMember(b) {
				level = "Public"
				break
			}
		}
		out = append(out, schema.BucketACLEntry{
			Container: name,
			Level:     level,
		})
	}
	return out, nil
}

// ExposeBucket grants `allUsers:objectViewer` on the bucket — the GCS
// equivalent of "public-read".
func (d *Driver) ExposeBucket(ctx context.Context, container, level string) (string, error) {
	if d == nil || d.Client == nil {
		return "", errors.New("gcp storage: nil api client")
	}
	policy, err := d.getBucketIamPolicy(ctx, container)
	if err != nil {
		return "", err
	}
	policy.Bindings = upsertBinding(policy.Bindings, objectViewerRole, allUsersMember)
	if err := d.setBucketIamPolicy(ctx, container, policy); err != nil {
		return "", err
	}
	return "Public", nil
}

// UnexposeBucket removes `allUsers` from any role granting public read access.
func (d *Driver) UnexposeBucket(ctx context.Context, container string) error {
	if d == nil || d.Client == nil {
		return errors.New("gcp storage: nil api client")
	}
	policy, err := d.getBucketIamPolicy(ctx, container)
	if err != nil {
		return err
	}
	policy.Bindings = removeMember(policy.Bindings, allUsersMember)
	return d.setBucketIamPolicy(ctx, container, policy)
}

func (d *Driver) getBucketIamPolicy(ctx context.Context, bucket string) (api.GCSPolicy, error) {
	var resp api.GCSPolicy
	err := d.Client.Do(ctx, api.Request{
		Method:     http.MethodGet,
		BaseURL:    api.StorageBaseURL,
		Path:       fmt.Sprintf("/storage/v1/b/%s/iam", url.PathEscape(bucket)),
		Idempotent: true,
	}, &resp)
	return resp, err
}

func (d *Driver) setBucketIamPolicy(ctx context.Context, bucket string, policy api.GCSPolicy) error {
	body, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	return d.Client.Do(ctx, api.Request{
		Method:  http.MethodPut,
		BaseURL: api.StorageBaseURL,
		Path:    fmt.Sprintf("/storage/v1/b/%s/iam", url.PathEscape(bucket)),
		Body:    body,
	}, nil)
}

func upsertBinding(bindings []api.GCSPolicyBind, role, member string) []api.GCSPolicyBind {
	for i, b := range bindings {
		if b.Role == role {
			for _, existing := range b.Members {
				if existing == member {
					return bindings
				}
			}
			bindings[i].Members = append(bindings[i].Members, member)
			return bindings
		}
	}
	return append(bindings, api.GCSPolicyBind{Role: role, Members: []string{member}})
}

func removeMember(bindings []api.GCSPolicyBind, member string) []api.GCSPolicyBind {
	out := make([]api.GCSPolicyBind, 0, len(bindings))
	for _, b := range bindings {
		filtered := make([]string, 0, len(b.Members))
		for _, m := range b.Members {
			if m == member {
				continue
			}
			filtered = append(filtered, m)
		}
		if len(filtered) == 0 {
			continue
		}
		b.Members = filtered
		out = append(out, b)
	}
	return out
}

func isPublicMember(b api.GCSPolicyBind) bool {
	for _, m := range b.Members {
		if m == allUsersMember || m == "allAuthenticatedUsers" {
			return true
		}
	}
	return false
}
