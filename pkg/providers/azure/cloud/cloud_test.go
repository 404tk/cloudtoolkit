package cloud

import (
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
)

func TestForChina(t *testing.T) {
	endpoints := For(auth.CloudChina)
	if endpoints.ActiveDirectory != "https://login.chinacloudapi.cn/" {
		t.Fatalf("unexpected AD endpoint: %s", endpoints.ActiveDirectory)
	}
	if endpoints.ResourceManager != "https://management.chinacloudapi.cn/" {
		t.Fatalf("unexpected RM endpoint: %s", endpoints.ResourceManager)
	}
}
