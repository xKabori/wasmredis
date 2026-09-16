package wasmredis

import (
	"sort"
	"strings"
)

func (e *Engine) addToIndex(value string, key string) {
	if e.index[value] == nil {
		e.index[value] = make(map[string]bool)
	}
	e.index[value][key] = true
}

func (e *Engine) removeFromIndex(value string, key string) {
	delete(e.index[value], key)
	if len(e.index[value]) == 0 {
		delete(e.index, value)
	}
}

func (e *Engine) GetEquals(value string) []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	var keys []string
	for key := range e.index[value] {
		if !e.expireIfNeeded(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func (e *Engine) GetContains(substring string) []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	var keys []string
	for key, entry := range e.state {
		if e.expireIfNeeded(key) {
			continue
		}
		if strings.Contains(entry.Value, substring) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
