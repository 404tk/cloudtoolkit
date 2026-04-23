package replay

import (
	"net/http"
	"sync"
)

var (
	replayMu     sync.Mutex
	replayClient *http.Client
	replayState  *transport
)

func replayHTTPClient() *http.Client {
	replayMu.Lock()
	defer replayMu.Unlock()
	if replayClient == nil {
		replayState = newTransport()
		replayClient = &http.Client{Transport: replayState}
	}
	return replayClient
}

func Reset() {
	replayMu.Lock()
	defer replayMu.Unlock()
	replayClient = nil
	replayState = nil
}
