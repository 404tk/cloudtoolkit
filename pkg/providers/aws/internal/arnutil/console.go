package arnutil

import "strings"

func ConsoleURLForARN(accountARN string) string {
	parts := strings.Split(accountARN, ":")
	if len(parts) <= 4 || parts[4] == "" {
		return ""
	}
	if parts[1] == "aws-cn" {
		return "https://" + parts[4] + ".signin.amazonaws.cn/console"
	}
	return "https://" + parts[4] + ".signin.aws.amazon.com/console"
}
