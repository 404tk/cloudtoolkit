package auth

import (
	"context"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

func TestFromOptionsAndRetrieve(t *testing.T) {
	options := schema.Options{
		utils.AccessKey:     "ak",
		utils.SecretKey:     "sk",
		utils.SecurityToken: "token",
	}

	credential, err := FromOptions(options)
	if err != nil {
		t.Fatalf("FromOptions() error = %v", err)
	}
	creds, err := credential.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if creds.AccessKeyID != "ak" || creds.SecretAccessKey != "sk" || creds.SessionToken != "token" {
		t.Fatalf("unexpected credentials: %+v", creds)
	}
}

func TestValidateRejectsEmptyFields(t *testing.T) {
	if err := New("", "sk", "").Validate(); err == nil {
		t.Fatal("expected validation error for empty access key")
	}
	if err := New("ak", "", "").Validate(); err == nil {
		t.Fatal("expected validation error for empty secret key")
	}
}
