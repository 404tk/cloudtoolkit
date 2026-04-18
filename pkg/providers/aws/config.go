package aws

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/internal/arnutil"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awsv2config "github.com/aws/aws-sdk-go-v2/config"
	awsv2credentials "github.com/aws/aws-sdk-go-v2/credentials"
)

func newConfig(
	ctx context.Context,
	accessKey string,
	secretKey string,
	token string,
	region string,
	version string,
) (awsv2.Config, error) {
	return awsv2config.LoadDefaultConfig(
		ctx,
		awsv2config.WithRegion(resolveBootstrapRegion(region, version)),
		awsv2config.WithCredentialsProvider(
			awsv2credentials.NewStaticCredentialsProvider(accessKey, secretKey, token),
		),
	)
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
