package auth

import (
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

func TestFromOptionsChinaCloud(t *testing.T) {
	cred, err := FromOptions(schema.Options{
		utils.AzureClientId:       "client-id",
		utils.AzureClientSecret:   "client-secret",
		utils.AzureTenantId:       "tenant-id",
		utils.AzureSubscriptionId: "sub-id",
		utils.Version:             "China",
	})
	if err != nil {
		t.Fatalf("FromOptions returned error: %v", err)
	}
	if cred.Cloud != CloudChina {
		t.Fatalf("unexpected cloud: %q", cred.Cloud)
	}
	if cred.SubscriptionID != "sub-id" {
		t.Fatalf("unexpected subscription: %q", cred.SubscriptionID)
	}
}
