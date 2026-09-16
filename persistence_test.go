package wasmredis

import (
	"strings"
	"testing"
	"time"
)

func TestFlushWritesBufferToAOF(t *testing.T) {
	storage := NewFileStorage(t.TempDir())
	e := NewEngine(storage)
	e.Set("name", "matt")
	e.Delete("age")

	err := e.Flush()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	aof, err := storage.ReadAOF()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	lines := strings.Count(string(aof), "\n")
	if lines != 2 {
		t.Errorf("attendu 2 lignes dans l'AOF, obtenu %d : %q", lines, aof)
	}
	if len(e.buffer) != 0 {
		t.Errorf("buffer attendu vide après le flush, obtenu %d opérations", len(e.buffer))
	}
}

func TestFlushEmptyBufferWritesNothing(t *testing.T) {
	storage := NewFileStorage(t.TempDir())
	e := NewEngine(storage)
	e.Set("name", "matt")

	err := e.Flush()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	err = e.Flush()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	aof, err := storage.ReadAOF()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	lines := strings.Count(string(aof), "\n")
	if lines != 1 {
		t.Errorf("attendu 1 ligne dans l'AOF, obtenu %d : %q", lines, aof)
	}
}

func TestSnapshotWritesStateAndClearsAOF(t *testing.T) {
	storage := NewFileStorage(t.TempDir())
	e := NewEngine(storage)
	e.Set("name", "matt")
	err := e.Flush()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	e.Set("age", "30")

	err = e.Snapshot()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	snapshot, err := storage.ReadSnapshot()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if !strings.Contains(string(snapshot), "matt") || !strings.Contains(string(snapshot), "30") {
		t.Errorf("le snapshot devrait contenir tout le state, obtenu %q", snapshot)
	}

	aof, err := storage.ReadAOF()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if len(aof) != 0 {
		t.Errorf("AOF attendu vide après le snapshot, obtenu %q", aof)
	}
	if len(e.buffer) != 0 {
		t.Errorf("buffer attendu vide après le snapshot, obtenu %d opérations", len(e.buffer))
	}
}

func TestStartFlushesInBackground(t *testing.T) {
	storage := NewFileStorage(t.TempDir())
	e := NewEngine(storage)
	e.Start()
	e.Set("name", "matt")

	time.Sleep(1500 * time.Millisecond)

	aof, err := storage.ReadAOF()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if !strings.Contains(string(aof), "matt") {
		t.Errorf("l'AOF devrait contenir matt après ~1s, obtenu %q", aof)
	}
}

func TestRestoreFromAOF(t *testing.T) {
	dir := t.TempDir()

	e1 := NewEngine(NewFileStorage(dir))
	e1.Set("name", "matt")
	e1.Set("age", "30")
	e1.Set("temp", "x")
	e1.Delete("temp")
	e1.Set("age", "31")
	err := e1.Flush()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	e2 := NewEngine(NewFileStorage(dir))
	err = e2.Restore()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	assertSameState(t, e1, e2)
}

func TestRestoreFromSnapshotAndAOF(t *testing.T) {
	dir := t.TempDir()

	e1 := NewEngine(NewFileStorage(dir))
	e1.Set("name", "matt")
	e1.Set("age", "30")
	err := e1.Snapshot()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	e1.Set("age", "31")
	e1.Delete("name")
	e1.Set("city", "paris")
	err = e1.Flush()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	e2 := NewEngine(NewFileStorage(dir))
	err = e2.Restore()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	assertSameState(t, e1, e2)
}

func TestRestoreEmptyDir(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))

	err := e.Restore()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if len(e.state) != 0 {
		t.Errorf("state attendu vide, obtenu %d clés", len(e.state))
	}
}

func TestRestoreDoesNotLogOperations(t *testing.T) {
	dir := t.TempDir()

	e1 := NewEngine(NewFileStorage(dir))
	e1.Set("name", "matt")
	err := e1.Flush()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	e2 := NewEngine(NewFileStorage(dir))
	err = e2.Restore()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if len(e2.buffer) != 0 {
		t.Errorf("le restore ne doit rien mettre dans le buffer, obtenu %d opérations", len(e2.buffer))
	}
}

func assertSameState(t *testing.T, want *Engine, got *Engine) {
	t.Helper()
	if len(got.state) != len(want.state) {
		t.Fatalf("attendu %d clés, obtenu %d", len(want.state), len(got.state))
	}
	for key, wantEntry := range want.state {
		gotEntry, ok := got.state[key]
		if !ok || gotEntry.Value != wantEntry.Value || !gotEntry.ExpiresAt.Equal(wantEntry.ExpiresAt) {
			t.Errorf("clé %q : obtenu %+v, attendu %+v", key, gotEntry, wantEntry)
		}
	}
}
