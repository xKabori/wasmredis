package wasmredis

import (
	"slices"
	"testing"
)

func TestGetEquals(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Set("alice", "30")
	e.Set("bob", "30")
	e.Set("carol", "40")

	got := e.GetEquals("30")
	want := []string{"alice", "bob"}
	if !slices.Equal(got, want) {
		t.Errorf("obtenu %v, attendu %v", got, want)
	}
}

func TestGetEqualsNoMatch(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Set("alice", "30")

	got := e.GetEquals("99")
	if len(got) != 0 {
		t.Errorf("attendu aucune clé, obtenu %v", got)
	}
}

func TestGetEqualsAfterOverwrite(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Set("alice", "30")
	e.Set("alice", "40")

	old := e.GetEquals("30")
	if len(old) != 0 {
		t.Errorf("l'ancienne valeur ne doit plus être indexée, obtenu %v", old)
	}
	got := e.GetEquals("40")
	want := []string{"alice"}
	if !slices.Equal(got, want) {
		t.Errorf("obtenu %v, attendu %v", got, want)
	}
}

func TestGetEqualsAfterDelete(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Set("alice", "30")
	e.Set("bob", "30")
	e.Delete("alice")

	got := e.GetEquals("30")
	want := []string{"bob"}
	if !slices.Equal(got, want) {
		t.Errorf("obtenu %v, attendu %v", got, want)
	}

	e.Delete("bob")
	_, found := e.index["30"]
	if found {
		t.Errorf("l'index ne doit plus contenir la valeur 30 quand plus aucune clé ne la porte")
	}
}

func TestGetContains(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Set("name", "matt")
	e.Set("nickname", "matteo")
	e.Set("city", "paris")

	got := e.GetContains("att")
	want := []string{"name", "nickname"}
	if !slices.Equal(got, want) {
		t.Errorf("obtenu %v, attendu %v", got, want)
	}
}

func TestRunGetEquals(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Run("SET alice 30")
	e.Run("SET bob 30")
	e.Run("SET carol 40")

	value, err := e.Run("GET EQUALS 30")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if value != "alice\nbob" {
		t.Errorf("obtenu %q, attendu %q", value, "alice\nbob")
	}
}

func TestRunGetContains(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Run("SET name matt")
	e.Run("SET city paris")

	value, err := e.Run("get contains ari")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if value != "city" {
		t.Errorf("obtenu %q, attendu %q", value, "city")
	}
}

func TestIndexRebuiltAfterRestore(t *testing.T) {
	dir := t.TempDir()

	e1 := NewEngine(NewFileStorage(dir))
	e1.Set("alice", "30")
	err := e1.Snapshot()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	e1.Set("bob", "30")
	err = e1.Flush()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	e2 := NewEngine(NewFileStorage(dir))
	err = e2.Restore()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	got := e2.GetEquals("30")
	want := []string{"alice", "bob"}
	if !slices.Equal(got, want) {
		t.Errorf("obtenu %v, attendu %v", got, want)
	}
}
