package azure

import (
	"strings"
	"testing"
)

func TestUserManagementRejectsBareUsername(t *testing.T) {
	p := &Provider{}
	_, err := p.UserManagement("add", "ctk-user", "Password!2026")
	if err == nil {
		t.Fatal("expected bare username to be rejected")
	}
	if !strings.Contains(err.Error(), "full userPrincipalName") {
		t.Fatalf("unexpected error: %v", err)
	}
}
