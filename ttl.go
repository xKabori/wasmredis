package wasmredis

import "time"

type Entry struct {
	Value     string
	ExpiresAt time.Time
}

func (entry Entry) isExpired() bool {
	if entry.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(entry.ExpiresAt)
}

func (e *Engine) expireIfNeeded(key string) bool {
	entry, ok := e.state[key]
	if !ok || !entry.isExpired() {
		return false
	}
	e.removeKey(key)
	e.logOperation(Operation{Type: CmdDelete, Key: key})
	return true
}
