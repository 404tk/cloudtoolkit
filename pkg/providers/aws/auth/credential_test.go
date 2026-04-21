package auth

import (
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

func TestFromOptionsParsesCredential(t *testing.T) {
	options := schema.Options{
		utils.AccessKey:     "ak",
		utils.SecretKey:     "sk",
		utils.SecurityToken: "token",
	}

	credential, err := FromOptions(options)
	if err != nil {
		t.Fatalf("FromOptions() error = %v", err)
	}
	if credential.AccessKeyID != "ak" || credential.SecretAccessKey != "sk" || credential.SessionToken != "token" {
		t.Fatalf("unexpected credential: %+v", credential)
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
