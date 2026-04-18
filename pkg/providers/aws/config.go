package aws

import (
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/internal/arnutil"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
)

func newConfig(
	accessKey string,
	secretKey string,
	token string,
	region string,
	version string,
) (awsv2.Config, error) {
	credential := auth.New(accessKey, secretKey, token)
	if err := credential.Validate(); err != nil {
		return awsv2.Config{}, err
	}
	return awsv2.Config{
		Region:      resolveBootstrapRegion(region, version),
		Credentials: awsv2.NewCredentialsCache(credential),
	}, nil
}

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
