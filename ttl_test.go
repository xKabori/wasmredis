package wasmredis

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestTTLNotExpiredYet(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	_, err := e.Run(`SET name "matt" EX 60`)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	value, err := e.Run("GET name")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if value != "matt" {
		t.Errorf("obtenu %q, attendu %q", value, "matt")
	}
}

func TestTTLExpiredGet(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	_, err := e.Run("SET name matt EX 1")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	time.Sleep(1100 * time.Millisecond)

	_, err = e.Run("GET name")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("attendu ErrKeyNotFound, obtenu %v", err)
	}
	_, found := e.state["name"]
	if found {
		t.Errorf("la clé expirée doit avoir été supprimée du state")
	}
}

func TestTTLExpiredFilters(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.SetWithTTL("alice", "30", 1)
	e.Set("bob", "30")
	e.SetWithTTL("carol", "paris", 1)
	e.Set("dave", "paris")

	time.Sleep(1100 * time.Millisecond)

	got := e.GetEquals("30")
	want := []string{"bob"}
	if !slices.Equal(got, want) {
		t.Errorf("equals : obtenu %v, attendu %v", got, want)
	}

	got = e.GetContains("par")
	want = []string{"dave"}
	if !slices.Equal(got, want) {
		t.Errorf("contains : obtenu %v, attendu %v", got, want)
	}
}

func TestTTLKeptAfterRestore(t *testing.T) {
	dir := t.TempDir()

	e1 := NewEngine(NewFileStorage(dir))
	e1.SetWithTTL("session", "abc", 60)
	err := e1.Snapshot()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	e1.SetWithTTL("token", "xyz", 120)
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
