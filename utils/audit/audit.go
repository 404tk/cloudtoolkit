package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Record struct {
	Timestamp string `json:"timestamp"`
	Provider  string `json:"provider"`
	Operation string `json:"operation"`
	Target    string `json:"target,omitempty"`
	Args      string `json:"args,omitempty"`
}

var (
	mu   sync.Mutex
	path string
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	path = filepath.Join(home, ".config", "cloudtoolkit", "audit.log")
}

// Log appends one JSONL record to ~/.config/cloudtoolkit/audit.log. Errors are
// reported via logger but never surface to callers, so a misbehaving log disk
// cannot block a security-sensitive operation.
func Log(r Record) {
	if path == "" {
		return
	}
	if r.Timestamp == "" {
		r.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	b, err := json.Marshal(r)
	if err != nil {
		logger.Error("Audit marshal failed:", err)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		logger.Error("Audit mkdir failed:", err)
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		logger.Error("Audit open failed:", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		logger.Error("Audit write failed:", err)
	}
}
