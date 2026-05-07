package replay

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/runtime/vmexecspec"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// TestReplayE2E_NewCapabilities exercises the JDCloud payloads touched by the
// v3-A spec sync and replay list update. It validates each happy path through
// the public Provider methods so the replay transport, fixtures, and capability
// glue stay in sync — not just the per-driver httptest contracts.
func TestReplayE2E_NewCapabilities(t *testing.T) {
	provider := newReplayProvider(t)

	t.Run("event-check_dump", func(t *testing.T) {
		result, err := provider.EventDump(context.Background(), "dump", "")
		if err != nil {
			t.Fatalf("EventDump: %v", err)
		}
		if result.Action != "dump" {
			t.Errorf("expected action=dump, got %q", result.Action)
		}
		if len(result.Events) != 3 {
			t.Fatalf("expected 3 demo events, got %d", len(result.Events))
		}
		if result.Events[0].Id != "jdc-evt-0001" {
			t.Errorf("unexpected first event id: %+v", result.Events[0])
		}
	})

	t.Run("event-check_whitelist_unsupported", func(t *testing.T) {
		if _, err := provider.EventDump(context.Background(), "whitelist", ""); err == nil {
			t.Fatalf("expected ActionTrail whitelist to be unsupported")
		}
	})

	t.Run("rds-account-check_useradd_userdel", func(t *testing.T) {
		env.SetActiveForTest(t, &env.Env{
			RDSAccount: "ctkvalidator:VeryLongDemoPassword!9001",
			RunTimeout: time.Minute,
		})
		add, err := provider.DBManagement(context.Background(), "useradd", "rds-prod-01")
		if err != nil {
			t.Fatalf("useradd: %v", err)
		}
		if add.Action != "useradd" || add.Username != "ctkvalidator" {
			t.Errorf("unexpected add result: %+v", add)
		}
		del, err := provider.DBManagement(context.Background(), "userdel", "rds-prod-01")
		if err != nil {
			t.Fatalf("userdel: %v", err)
		}
		if del.Action != "userdel" || del.Username != "ctkvalidator" {
			t.Errorf("unexpected del result: %+v", del)
		}
	})

	t.Run("rds-account-check_invalid_action", func(t *testing.T) {
		if _, err := provider.DBManagement(context.Background(), "wipe", "rds-prod-01"); err == nil {
			t.Fatalf("expected error for unsupported action")
		}
	})

	t.Run("iam-credential-check_lifecycle", func(t *testing.T) {
		ctx := context.Background()
		const principal = "ctk-demo-readonly"
		listEmpty, err := provider.IAMCredential(ctx, "list", principal, "")
		if err != nil {
			t.Fatalf("list (initial): %v", err)
		}
		if len(listEmpty.Credentials) != 0 {
			t.Fatalf("expected zero keys initially, got %d", len(listEmpty.Credentials))
		}
		created, err := provider.IAMCredential(ctx, "create", principal, "")
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if created.CredentialID == "" || created.CredentialData == "" {
			t.Fatalf("expected create to surface AK + secret, got %+v", created)
		}
		listAfter, err := provider.IAMCredential(ctx, "list", principal, "")
		if err != nil {
			t.Fatalf("list (after create): %v", err)
		}
		if len(listAfter.Credentials) != 1 || listAfter.Credentials[0].CredentialID != created.CredentialID {
			t.Fatalf("expected 1 key matching %s, got %+v", created.CredentialID, listAfter.Credentials)
		}
		if _, err := provider.IAMCredential(ctx, "delete", principal, created.CredentialID); err != nil {
			t.Fatalf("delete: %v", err)
		}
		listFinal, err := provider.IAMCredential(ctx, "list", principal, "")
		if err != nil {
			t.Fatalf("list (after delete): %v", err)
		}
		if len(listFinal.Credentials) != 0 {
			t.Fatalf("expected zero keys after delete, got %d", len(listFinal.Credentials))
		}
	})

	t.Run("iam-credential-check_unknown_principal", func(t *testing.T) {
		if _, err := provider.IAMCredential(context.Background(), "list", "nonexistent-user", ""); err == nil {
			t.Fatalf("expected error listing keys on unknown sub user")
		}
	})

	t.Run("instance-cmd-check_headless_spec", func(t *testing.T) {
		result, err := provider.ExecuteCloudVMCommand(context.Background(), "i-jdc001", vmexecspec.BuildLinux("whoami"))
		if err != nil {
			t.Fatalf("ExecuteCloudVMCommand: %v", err)
		}
		if !strings.Contains(result.Output, "whoami") {
			t.Fatalf("expected replay output to include command, got %q", result.Output)
		}
	})
}

// TestReplayE2E_BadAccessKey verifies the auth check returns provider-style
// `InvalidAccessKeyId` rather than silently accepting wrong AK pairs.
func TestReplayE2E_BadAccessKey(t *testing.T) {
	options := schema.Options{
		utils.AccessKey: "JDC_AKLT_BOGUS_AK_FOR_TEST_REPLAY_X",
		utils.SecretKey: "JDCExampleSecretKeyValueDEMOreplay00000",
		utils.Region:    "cn-north-1",
	}
	provider, err := jdcloud.NewWithConfig(options, ClientConfig())
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	_, err = provider.EventDump(context.Background(), "dump", "")
	if err == nil {
		t.Fatalf("expected InvalidAccessKeyId error, got nil")
	}
	if !strings.Contains(err.Error(), "Access Key Id you provided does not exist") {
		t.Errorf("expected InvalidAccessKeyId-style error, got %q", err.Error())
	}
}

func newReplayProvider(t *testing.T) *jdcloud.Provider {
	t.Helper()
	options := schema.Options{
		utils.AccessKey: demoCredentials.AccessKey,
		utils.SecretKey: demoCredentials.SecretKey,
		utils.Region:    demoRegion,
	}
	provider, err := jdcloud.NewWithConfig(options, ClientConfig())
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	return provider
}
