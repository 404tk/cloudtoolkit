package logger

import (
	"io"
	"log"
	"os"
	"sync"
)

var (
	Debug   = true      // 是否输出信息日志
	info    *log.Logger // 阶段日志
	warning *log.Logger // 需要注意的信息
	_error  *log.Logger // 错误日志
	fatal   *log.Logger // 异常退出
	mu      sync.RWMutex
)

func init() {
	resetLoggers(os.Stdout, os.Stderr)
}

func resetLoggers(stdout, stderr io.Writer) {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	info = log.New(stdout, "[*] ", log.Ltime)
	warning = log.New(stdout, "[+] ", log.Ltime)
	_error = log.New(stderr, "[-] ", log.Ltime)
	fatal = log.New(stderr, "[!] ", log.Ltime)
}

func SetOutput(w io.Writer) {
	SetOutputs(w, w)
}

func SetOutputs(stdout, stderr io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	resetLoggers(stdout, stderr)
}

func SetDebug(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	Debug = enabled
}

func Info(v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	if Debug {
		info.Println(v...)
	}
}

func Warning(v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	warning.Println(v...)
}

func Error(v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	_error.Println(v...)
}

func Fatalf(format string, args ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fatal.Printf(format, args...)
	os.Exit(1)
}
