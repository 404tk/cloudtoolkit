package replay

import (
	"context"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// TestReplayE2E_IAMCredential exercises the iam-credential payload through the
// public Provider methods so the replay transport, fixtures, and capability
// glue stay in sync with what spec.go advertises after the v3-B sync.
func TestReplayE2E_IAMCredential(t *testing.T) {
	provider := newReplayProvider(t)
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

	if _, err := provider.IAMCredential(ctx, "list", "nonexistent-user", ""); err == nil {
		t.Fatalf("expected error listing keys on unknown user")
	}
}

// TestReplayE2E_BadAccessKey verifies that swapping in a bogus AccessKey lands
// in UCloud's `Invalid PublicKey` failure path rather than silently accepting
// the request.
func TestReplayE2E_BadAccessKey(t *testing.T) {
	options := schema.Options{
		utils.AccessKey: "ucloudpubkey-BOGUS-NEVER-VALID",
		utils.SecretKey: "UCloudzTjmbGC4GR3dbgueU3zRZ7i43eTYQc3EZYqoFR",
		utils.Region:    "cn-bj2",
	}
	if _, err := ucloud.NewWithConfig(options, ClientConfig()); err == nil {
		t.Fatalf("expected NewWithConfig to fail with bogus access key")
	} else if !strings.Contains(err.Error(), "Invalid PublicKey") {
		t.Errorf("expected `Invalid PublicKey` in error, got %q", err.Error())
	}
}

func newReplayProvider(t *testing.T) *ucloud.Provider {
	t.Helper()
	options := schema.Options{
		utils.AccessKey: demoCredentials.AccessKey,
		utils.SecretKey: demoCredentials.SecretKey,
		utils.Region:    "cn-bj2",
	}
	provider, err := ucloud.NewWithConfig(options, ClientConfig())
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	return provider
}
