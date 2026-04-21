package replay

import "sync"

type replayState struct {
	Active   bool
	Provider string
}

var (
	replayStateMu sync.RWMutex
	currentState  replayState
)

func Enable(provider string) {
	replayStateMu.Lock()
	defer replayStateMu.Unlock()
	currentState = replayState{
		Active:   true,
		Provider: normalizeProvider(provider),
	}
}

func Disable() {
	replayStateMu.Lock()
	defer replayStateMu.Unlock()
	currentState = replayState{}
}

func IsActive() bool {
	replayStateMu.RLock()
	defer replayStateMu.RUnlock()
	return currentState.Active && currentState.Provider != ""
}

func ActiveProvider() string {
	replayStateMu.RLock()
	defer replayStateMu.RUnlock()
	return currentState.Provider
}

func IsActiveForProvider(provider string) bool {
	replayStateMu.RLock()
	defer replayStateMu.RUnlock()
	return currentState.Active && currentState.Provider != "" && currentState.Provider == normalizeProvider(provider)
}
