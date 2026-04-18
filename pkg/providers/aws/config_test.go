package aws

import (
	"context"
	"testing"
)

func TestResolveBootstrapRegion(t *testing.T) {
	tests := []struct {
		name    string
		region  string
		version string
		want    string
	}{
		{name: "explicit", region: "ap-southeast-1", version: "", want: "ap-southeast-1"},
		{name: "all global", region: "all", version: "", want: "us-east-1"},
		{name: "all china", region: "all", version: "China", want: "cn-northwest-1"},
		{name: "empty global", region: "", version: "", want: "us-east-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveBootstrapRegion(tt.region, tt.version); got != tt.want {
				t.Fatalf("resolveBootstrapRegion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCurrentUserNameFromARN(t *testing.T) {
	tests := []struct {
		arn  string
		want string
	}{
		{arn: "arn:aws:iam::123456789012:root", want: "root"},
		{arn: "arn:aws:sts::123456789012:assumed-role/admin/session", want: "session"},
		{arn: "arn:aws:iam::123456789012:user/demo", want: "demo"},
	}
	for _, tt := range tests {
		if got := currentUserNameFromARN(tt.arn); got != tt.want {
			t.Fatalf("currentUserNameFromARN(%q) = %q, want %q", tt.arn, got, tt.want)
		}
	}
}

func TestConsoleURLForARN(t *testing.T) {
	tests := []struct {
		arn  string
		want string
	}{
		{arn: "arn:aws:iam::123456789012:user/demo", want: "https://123456789012.signin.aws.amazon.com/console"},
		{arn: "arn:aws-cn:iam::210987654321:user/demo", want: "https://210987654321.signin.amazonaws.cn/console"},
		{arn: "invalid", want: ""},
	}
	for _, tt := range tests {
		if got := consoleURLForARN(tt.arn); got != tt.want {
			t.Fatalf("consoleURLForARN(%q) = %q, want %q", tt.arn, got, tt.want)
		}
	}
}

func TestNewConfigUsesStaticCredentials(t *testing.T) {
	cfg, err := newConfig("ak", "sk", "token", "all", "China")
	if err != nil {
		t.Fatalf("newConfig() error = %v", err)
	}
	if cfg.Region != "cn-northwest-1" {
		t.Fatalf("unexpected region: %s", cfg.Region)
	}
	if cfg.Credentials == nil {
		t.Fatal("expected credentials provider")
	}
	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if creds.AccessKeyID != "ak" || creds.SecretAccessKey != "sk" || creds.SessionToken != "token" {
		t.Fatalf("unexpected credentials: %+v", creds)
	}
}
