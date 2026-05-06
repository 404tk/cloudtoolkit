package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func newAccessKeysTestDriver(baseURL string) *Driver {
	return &Driver{
		Client: api.NewClient(
			auth.New("AKID", "SECRET", ""),
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		),
		Region: "us-east-1",
	}
}

func parseAccessKeysForm(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatalf("ParseForm: %v", err)
	}
	return r.PostForm
}

func TestListAccessKeysParsesMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := parseAccessKeysForm(t, r)
		if got := values.Get("Action"); got != "ListAccessKeys" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := values.Get("UserName"); got != "alice" {
			t.Fatalf("unexpected user: %s", got)
		}
		_, _ = w.Write([]byte(`
<ListAccessKeysResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <ListAccessKeysResult>
    <AccessKeyMetadata>
      <member>
        <UserName>alice</UserName>
        <AccessKeyId>AKIAIOSFODNN7EXAMPLE001</AccessKeyId>
        <Status>Active</Status>
        <CreateDate>2026-01-01T10:00:00Z</CreateDate>
      </member>
      <member>
        <UserName>alice</UserName>
        <AccessKeyId>AKIAIOSFODNN7EXAMPLE002</AccessKeyId>
        <Status>Inactive</Status>
        <CreateDate>2026-02-15T08:30:00Z</CreateDate>
      </member>
    </AccessKeyMetadata>
    <IsTruncated>false</IsTruncated>
  </ListAccessKeysResult>
  <ResponseMetadata>
    <RequestId>req-list-access-keys</RequestId>
  </ResponseMetadata>
</ListAccessKeysResponse>`))
	}))
	defer server.Close()

	driver := newAccessKeysTestDriver(server.URL)
	got, err := driver.ListAccessKeys(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListAccessKeys: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(got))
	}
	if got[0].CredentialID != "AKIAIOSFODNN7EXAMPLE001" || got[0].CredentialType != "Active" {
		t.Errorf("unexpected first key: %+v", got[0])
	}
	if got[1].CredentialType != "Inactive" {
		t.Errorf("unexpected second key status: %+v", got[1])
	}
	if got[0].ValidAfter != "2026-01-01T10:00:00Z" {
		t.Errorf("unexpected ValidAfter: %s", got[0].ValidAfter)
	}
}

func TestCreateAccessKeyReturnsSecret(t *testing.T) {
	var captured url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = parseAccessKeysForm(t, r)
		_, _ = w.Write([]byte(`
<CreateAccessKeyResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <CreateAccessKeyResult>
    <AccessKey>
      <UserName>alice</UserName>
      <AccessKeyId>AKIAIOSFODNN7MINTED01</AccessKeyId>
      <SecretAccessKey>wJalrXUtnFEMI/K7MDENG/CTKMINTSECRET01</SecretAccessKey>
      <Status>Active</Status>
      <CreateDate>2026-04-30T08:00:00Z</CreateDate>
    </AccessKey>
  </CreateAccessKeyResult>
  <ResponseMetadata>
    <RequestId>req-create-access-key</RequestId>
  </ResponseMetadata>
</CreateAccessKeyResponse>`))
	}))
	defer server.Close()

	driver := newAccessKeysTestDriver(server.URL)
	cred, secret, err := driver.CreateAccessKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}
	if captured.Get("Action") != "CreateAccessKey" {
		t.Errorf("unexpected action: %s", captured.Get("Action"))
	}
	if captured.Get("UserName") != "alice" {
		t.Errorf("unexpected user: %s", captured.Get("UserName"))
	}
	if cred.CredentialID != "AKIAIOSFODNN7MINTED01" {
		t.Errorf("unexpected key id: %+v", cred)
	}
	if secret != "wJalrXUtnFEMI/K7MDENG/CTKMINTSECRET01" {
		t.Errorf("unexpected secret: %s", secret)
	}
}

func TestCreateAccessKeyRejectsEmptyPrincipal(t *testing.T) {
	driver := newAccessKeysTestDriver("http://example.invalid")
	if _, _, err := driver.CreateAccessKey(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestDeleteAccessKeySendsExpectedForm(t *testing.T) {
	var captured url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = parseAccessKeysForm(t, r)
		_, _ = w.Write([]byte(`<DeleteAccessKeyResponse><ResponseMetadata><RequestId>r1</RequestId></ResponseMetadata></DeleteAccessKeyResponse>`))
	}))
	defer server.Close()

	driver := newAccessKeysTestDriver(server.URL)
	if err := driver.DeleteAccessKey(context.Background(), "alice", "AKIAIOSFODNN7EXAMPLE001"); err != nil {
		t.Fatalf("DeleteAccessKey: %v", err)
	}
	if captured.Get("Action") != "DeleteAccessKey" {
		t.Errorf("unexpected action: %s", captured.Get("Action"))
	}
	if captured.Get("AccessKeyId") != "AKIAIOSFODNN7EXAMPLE001" {
		t.Errorf("unexpected key id: %s", captured.Get("AccessKeyId"))
	}
	if captured.Get("UserName") != "alice" {
		t.Errorf("unexpected user: %s", captured.Get("UserName"))
	}
}

func TestDeleteAccessKeyPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`
<ErrorResponse>
  <Error>
    <Type>Sender</Type>
    <Code>NoSuchEntity</Code>
    <Message>The Access Key with id AKIA... cannot be found</Message>
  </Error>
  <RequestId>r1</RequestId>
</ErrorResponse>`))
	}))
	defer server.Close()

	driver := newAccessKeysTestDriver(server.URL)
	err := driver.DeleteAccessKey(context.Background(), "alice", "AKIAINVALID")
	if err == nil {
		t.Fatalf("expected error from DeleteAccessKey")
	}
	if !strings.Contains(err.Error(), "NoSuchEntity") {
		t.Errorf("expected NoSuchEntity in error, got %v", err)
	}
}

func TestDeleteAccessKeyRejectsEmptyID(t *testing.T) {
	driver := newAccessKeysTestDriver("http://example.invalid")
	if err := driver.DeleteAccessKey(context.Background(), "alice", "  "); err == nil {
		t.Fatalf("expected error for empty access key id")
	}
}
