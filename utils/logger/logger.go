package logger

import (
	"log"
	"os"
)

var (
	Debug   = true      // 是否输出log
	info    *log.Logger // 阶段日志
	warning *log.Logger // 需要注意的信息
	_error  *log.Logger // 错误日志
	fatal   *log.Logger // 异常退出
)

func init() {
	info = log.New(os.Stdout,
		"[*] ", 2)

	warning = log.New(os.Stdout,
		"[+] ", 2)

	_error = log.New(os.Stdout,
		"[-] ", 2)

	fatal = log.New(os.Stdout,
		"[x] ", 2)

}

func Info(v ...interface{}) {
	if Debug {
		info.Println(v...)
	}
}

func Warning(v ...interface{}) {
	if Debug {
		warning.Println(v...)
	}
}

func Error(v ...interface{}) {
	if Debug {
		_error.Println(v...)
	}
}

func Fatalf(format string, args ...interface{}) {
	fatal.Printf(format, args...)
	os.Exit(1)
}
