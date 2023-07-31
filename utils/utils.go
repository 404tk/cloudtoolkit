package utils

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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

func HttpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func CheckLogDir() {
	if v, _ := filepath.Glob(LogDir); len(v) == 0 {
		os.Mkdir(LogDir, os.ModePerm)
	}
}

func ParseBytes(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%v bytes", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(1024))
	case size < 1024*1024*1024:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(1024*1024))
	case size < 1024*1024*1024*1024:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(1024*1024*1024))
	case size < 1024*1024*1024*1024*1024:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(1024*1024*1024*1024))
	default:
		return fmt.Sprintf("%.2f PB", float64(size)/float64(1024*1024*1024*1024*1024))
	}
}

func WriteLog(filename string, msg string) {
	var text = []byte(msg + "\n")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	_, err = f.Write(text)
	if err != nil {
		return
	}
	f.Close()
}
