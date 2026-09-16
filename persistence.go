package wasmredis

import (
	"encoding/json"
	"strings"
	"time"
)

type Operation struct {
	Type      string
	Key       string
	Value     string
	ExpiresAt time.Time
}

func (e *Engine) logOperation(op Operation) {
	e.buffer = append(e.buffer, op)
}

func (e *Engine) Flush() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.buffer) == 0 {
		return nil
	}

	var text strings.Builder
	for _, op := range e.buffer {
		line, err := json.Marshal(op)
		if err != nil {
			return err
		}
		text.Write(line)
		text.WriteString("\n")
	}

	err := e.storage.AppendAOF([]byte(text.String()))
	if err != nil {
		return err
	}
	e.buffer = nil
	return nil
}

func (e *Engine) Snapshot() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, err := json.Marshal(e.state)
	if err != nil {
		return err
	}
	err = e.storage.WriteSnapshot(data)
	if err != nil {
		return err
	}
	err = e.storage.ClearAOF()
	if err != nil {
		return err
	}
	e.buffer = nil
	return nil
}

func (e *Engine) Restore() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	snapshotData, err := e.storage.ReadSnapshot()
	if err != nil {
		return err
	}
	if len(snapshotData) > 0 {
		var snapshot map[string]Entry
		err = json.Unmarshal(snapshotData, &snapshot)
		if err != nil {
			return err
		}
		for key, entry := range snapshot {
			e.putEntry(key, entry)
		}
	}

	aofData, err := e.storage.ReadAOF()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(aofData), "\n") {
		if line == "" {
			continue
		}
		var op Operation
		err = json.Unmarshal([]byte(line), &op)
		if err != nil {
			return err
		}
		e.applyOperation(op)
	}
	return nil
}

func (e *Engine) applyOperation(op Operation) {
	switch op.Type {
	case CmdSet:
		e.putEntry(op.Key, Entry{Value: op.Value, ExpiresAt: op.ExpiresAt})
	case CmdDelete:
		e.removeKey(op.Key)
	}
}

func (e *Engine) Start() {
	go e.flushLoop()
	go e.snapshotLoop()
}

func (e *Engine) flushLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		e.Flush()
	}
}

func (e *Engine) snapshotLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	for range ticker.C {
		e.Snapshot()
	}
}
