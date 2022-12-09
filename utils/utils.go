package utils

import (
	"crypto/md5"
	"fmt"
	"strings"
)

func ParseCmd(s string) (cmd string, args []string) {
	items := strings.Split(s, " ")
	cmd = items[0]
	if strings.HasPrefix(s, "set ") && len(items) > 2 {
		args = []string{items[1], strings.Join(items[2:], " ")}
	} else if len(items) > 1 {
		args = items[1:]
	}
	return
}

func Md5Encode(s string) string {
	data := []byte(s)
	has := md5.Sum(data)
	return fmt.Sprintf("%x", has)
}
