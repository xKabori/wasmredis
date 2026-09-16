package wasmredis

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrKeyNotFound = errors.New("clé absente")

type Engine struct {
	mu sync.Mutex

	state map[string]Entry

	index map[string]map[string]bool

	buffer []Operation

	storage Storage
}

func NewEngine(storage Storage) *Engine {
	return &Engine{
		state:   make(map[string]Entry),
		index:   make(map[string]map[string]bool),
		storage: storage,
	}
}

func (e *Engine) Set(key string, value string) {
	e.SetWithTTL(key, value, 0)
}

func (e *Engine) SetWithTTL(key string, value string, ttlSeconds int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	entry := Entry{Value: value}
	if ttlSeconds > 0 {
		entry.ExpiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	e.putEntry(key, entry)
	e.logOperation(Operation{Type: CmdSet, Key: key, Value: value, ExpiresAt: entry.ExpiresAt})
}

func (e *Engine) Get(key string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.expireIfNeeded(key)

	entry, ok := e.state[key]
	if !ok {
		return "", ErrKeyNotFound
	}
	return entry.Value, nil
}

func (e *Engine) Delete(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.removeKey(key)
	e.logOperation(Operation{Type: CmdDelete, Key: key})
}

func (e *Engine) Execute(cmd Command) (string, error) {
	switch cmd.Name {
	case CmdSet:
		e.SetWithTTL(cmd.Key, cmd.Value, cmd.TTLSeconds)
		return "OK", nil
	case CmdGet:
		if cmd.Filter == FilterEquals {
			return strings.Join(e.GetEquals(cmd.Value), "\n"), nil
		}
		if cmd.Filter == FilterContains {
			return strings.Join(e.GetContains(cmd.Value), "\n"), nil
		}
		return e.Get(cmd.Key)
	case CmdDelete:
		e.Delete(cmd.Key)
		return "OK", nil
	default:
		return "", ErrUnknownCommand
	}
}

func (e *Engine) Run(input string) (string, error) {
	cmd, err := ParseCommand(input)
	if err != nil {
		return "", err
	}
	return e.Execute(cmd)
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (e *Engine) GetAll() []KeyValue {
	e.mu.Lock()
	defer e.mu.Unlock()

	var keys []string
	for key := range e.state {
		if !e.expireIfNeeded(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	entries := make([]KeyValue, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, KeyValue{Key: key, Value: e.state[key].Value})
	}
	return entries
}

func (e *Engine) putEntry(key string, entry Entry) {
	e.removeKey(key)
	e.state[key] = entry
	e.addToIndex(entry.Value, key)
}

func (e *Engine) removeKey(key string) {
	oldEntry, ok := e.state[key]
	if !ok {
		return
	}
	delete(e.state, key)
	e.removeFromIndex(oldEntry.Value, key)
}
