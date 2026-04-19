package auth

import (
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

func TestFromOptionsChina(t *testing.T) {
	cred, err := FromOptions(schema.Options{
		utils.AccessKey: "AKID",
		utils.SecretKey: "SECRET",
		utils.Region:    "cn-north-4",
		utils.Version:   "China",
	})
	if err != nil {
		t.Fatalf("FromOptions() error = %v", err)
	}
	if cred.AK != "AKID" || cred.SK != "SECRET" || cred.Region != "cn-north-4" || cred.Intl {
		t.Fatalf("unexpected credential: %+v", cred)
	}
}

func TestFromOptionsIntl(t *testing.T) {
	cred, err := FromOptions(schema.Options{
		utils.AccessKey: "AKID",
		utils.SecretKey: "SECRET",
		utils.Region:    "ap-southeast-3",
		utils.Version:   "International",
	})
	if err != nil {
		t.Fatalf("FromOptions() error = %v", err)
	}
	if !cred.Intl {
		t.Fatalf("expected intl credential: %+v", cred)
	}
}

func TestCredentialValidate(t *testing.T) {
	if err := New("AKID", "SECRET", "cn-north-4", false).Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := New("", "SECRET", "cn-north-4", false).Validate(); err == nil {
		t.Fatal("expected empty access key error")
	}
	if err := New("AKID", "", "cn-north-4", false).Validate(); err == nil {
		t.Fatal("expected empty secret key error")
	}
}
