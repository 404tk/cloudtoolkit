package aws

import (
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/internal/arnutil"
)

func resolveBootstrapRegion(region string, version string) string {
	if region != "" && region != "all" {
		return region
	}
	if version == "China" {
		return "cn-northwest-1"
	}
	return "us-east-1"
}

func currentUserNameFromARN(accountARN string) string {
	if strings.HasSuffix(accountARN, "root") {
		return "root"
	}
	parts := strings.Split(accountARN, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return accountARN
}

func consoleURLForARN(accountARN string) string {
	return arnutil.ConsoleURLForARN(accountARN)
}
