package api

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
)

func sortedParamKeys(params map[string]string) []string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func signingPayload(params map[string]string) string {
	keys := sortedParamKeys(params)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString(params[key])
	}
	return builder.String()
}

func signature(params map[string]string, secretKey string) string {
	sum := sha1.Sum([]byte(signingPayload(params) + secretKey))
	return hex.EncodeToString(sum[:])
}
