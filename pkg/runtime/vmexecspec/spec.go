package vmexecspec

import (
	"encoding/base64"
	"strings"
)

const (
	linuxPrefix   = "__ctk_headless_sh__:"
	windowsPrefix = "__ctk_headless_cmd__:"
)

func BuildLinux(command string) string {
	return linuxPrefix + base64.StdEncoding.EncodeToString([]byte(command))
}

func BuildWindows(command string) string {
	return windowsPrefix + base64.StdEncoding.EncodeToString([]byte(command))
}

func Parse(spec string) (osType, command string, ok bool) {
	switch {
	case strings.HasPrefix(spec, linuxPrefix):
		return decode("linux", strings.TrimPrefix(spec, linuxPrefix))
	case strings.HasPrefix(spec, windowsPrefix):
		return decode("windows", strings.TrimPrefix(spec, windowsPrefix))
	default:
		return "", "", false
	}
}

func decode(osType, encoded string) (string, string, bool) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", "", false
	}
	return osType, string(raw), true
}
