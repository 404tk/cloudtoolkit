// Package logger emits stage-level diagnostic messages from CTK in two
// interchangeable formats:
//
//   - text (default): human-friendly "[*] HH:MM:SS message" lines, identical
//     to the legacy log.Logger output that the rest of the codebase already
//     produces.
//   - json: one JSON object per line, suitable for SIEM ingestion. Every
//     record carries any attributes attached via SetGlobalAttrs (run_id,
//     scenario, account, region, ...) so detection engineers can correlate
//     CTK validation actions with downstream telemetry.
//
// The active format is selected by `common.log_format` in config.yaml and
// applied during runner.InitConfig.
package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

// Format selects the output handler.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

var (
	mu sync.RWMutex

	// text driver
	info, warning, errorL *log.Logger

	// json driver — both go to the same writer pair as the text driver.
	// In JSON mode info/warning land on stdoutW, error lands on stderrW.
	jsonOut, jsonErr *slog.Logger

	stdoutW, stderrW io.Writer

	format    = FormatText
	debugFlag atomic.Bool
	baseAttrs []slog.Attr
)

func init() {
	debugFlag.Store(true)
	resetWriters(os.Stdout, os.Stderr)
}

// resetWriters rebuilds both text and json drivers using the supplied writers.
// nil arguments fall back to os.Stdout / os.Stderr (preserves legacy
// SetOutput(nil) "reset" semantics relied on by ~30 tests).
func resetWriters(stdout, stderr io.Writer) {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	stdoutW = stdout
	stderrW = stderr

	info = log.New(stdout, "[*] ", log.Ltime)
	warning = log.New(stdout, "[+] ", log.Ltime)
	errorL = log.New(stderr, "[-] ", log.Ltime)

	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	jsonOut = slog.New(slog.NewJSONHandler(stdout, opts))
	jsonErr = slog.New(slog.NewJSONHandler(stderr, opts))
}

// SetFormat switches the active output handler. Unknown values fall back to
// text and emit a warning so a typo in config.yaml degrades safely.
func SetFormat(f Format) {
	mu.Lock()
	switch f {
	case FormatJSON:
		format = FormatJSON
		mu.Unlock()
	case FormatText, "":
		format = FormatText
		mu.Unlock()
	default:
		format = FormatText
		mu.Unlock()
		Warning(fmt.Sprintf("logger: unknown log_format %q, falling back to text", string(f)))
	}
}

// CurrentFormat reports the currently active output format.
func CurrentFormat() Format {
	mu.RLock()
	defer mu.RUnlock()
	return format
}

// SetOutput overrides both stdout and stderr to the same writer. Passing nil
// restores os.Stdout / os.Stderr (preserves legacy behavior).
func SetOutput(w io.Writer) {
	SetOutputs(w, w)
}

// SetOutputs separately overrides stdout and stderr. Either argument may be
// nil to fall back to OS defaults.
func SetOutputs(stdout, stderr io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	resetWriters(stdout, stderr)
}

// SetDebug toggles whether Info-level records are emitted (legacy semantic).
func SetDebug(enabled bool) {
	debugFlag.Store(enabled)
}

// IsDebug reports whether Info-level emission is currently enabled.
func IsDebug() bool {
	return debugFlag.Load()
}

// SetGlobalAttrs replaces the slice of attributes attached to every emitted
// record. The headless layer pins run_id / provider / account / region here
// so JSON output gets a stable correlation envelope and text output gets
// trailing `key=value` hints.
//
// Pass no arguments to clear.
func SetGlobalAttrs(attrs ...slog.Attr) {
	mu.Lock()
	defer mu.Unlock()
	if len(attrs) == 0 {
		baseAttrs = nil
		return
	}
	baseAttrs = append(baseAttrs[:0], attrs...)
}

// Attrs converts alternating key / value pairs into a slog.Attr slice for
// SetGlobalAttrs convenience. Panics on odd argument count or non-string keys
// (matches slog's own contract).
func Attrs(kv ...any) []slog.Attr {
	if len(kv)%2 != 0 {
		panic("logger.Attrs: odd number of arguments")
	}
	out := make([]slog.Attr, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			panic("logger.Attrs: expected string key")
		}
		out = append(out, slog.Any(key, kv[i+1]))
	}
	return out
}

// Info emits a stage-level record. Suppressed when debug is off.
func Info(v ...interface{}) {
	if !debugFlag.Load() {
		return
	}
	emit(slog.LevelInfo, v...)
}

// Warning emits a notable-but-non-fatal record.
func Warning(v ...interface{}) {
	emit(slog.LevelWarn, v...)
}

// Error emits an error record. Always written to the stderr writer.
func Error(v ...interface{}) {
	emit(slog.LevelError, v...)
}

func emit(level slog.Level, args ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()

	if format == FormatJSON {
		// Sprintln matches log.Println's argument joining (spaces between all
		// operands), then trim its trailing newline so JSON msg fields stay
		// single-line.
		msg := strings.TrimRight(fmt.Sprintln(args...), "\n")
		l := jsonOut
		if level == slog.LevelError {
			l = jsonErr
		}
		l.LogAttrs(context.Background(), level, msg, baseAttrs...)
		return
	}

	var l *log.Logger
	switch level {
	case slog.LevelError:
		l = errorL
	case slog.LevelWarn:
		l = warning
	default:
		l = info
	}
	if len(baseAttrs) == 0 {
		l.Println(args...)
		return
	}
	// With pinned attrs, render via Sprintln so we can append the kv tail.
	msg := strings.TrimRight(fmt.Sprintln(args...), "\n")
	l.Println(buildTextLine(msg))
}

// buildTextLine appends global attrs as ` key=value` pairs to the message so
// text consumers tailing logs still see correlation hints when SetGlobalAttrs
// has been used. Returns msg unchanged when no attrs are pinned.
func buildTextLine(msg string) string {
	if len(baseAttrs) == 0 {
		return msg
	}
	var b strings.Builder
	b.Grow(len(msg) + 16*len(baseAttrs))
	b.WriteString(msg)
	for _, a := range baseAttrs {
		b.WriteByte(' ')
		b.WriteString(a.Key)
		b.WriteByte('=')
		b.WriteString(a.Value.String())
	}
	return b.String()
}
