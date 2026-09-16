package wasmredis

import (
	"errors"
	"testing"
)

func TestEngineSetGet(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Set("name", "matt")

	value, err := e.Get("name")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if value != "matt" {
		t.Errorf("obtenu %q, attendu %q", value, "matt")
	}
}

func TestEngineSetOverwrite(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Set("name", "matt")
	e.Set("name", "bob")

	value, err := e.Get("name")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if value != "bob" {
		t.Errorf("obtenu %q, attendu %q", value, "bob")
	}
}

func TestEngineGetMissingKey(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))

	_, err := e.Get("name")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("attendu ErrKeyNotFound, obtenu %v", err)
	}
}

func TestEngineDelete(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Set("name", "matt")
	e.Delete("name")

	_, err := e.Get("name")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("attendu ErrKeyNotFound, obtenu %v", err)
	}
}

func TestRunSetThenGet(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))

	_, err := e.Run(`SET msg "hello world"`)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	value, err := e.Run("GET msg")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if value != "hello world" {
		t.Errorf("obtenu %q, attendu %q", value, "hello world")
	}
}

func TestRunDeleteThenGet(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Run("SET name matt")
	e.Run("DELETE name")

	_, err := e.Run("GET name")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("attendu ErrKeyNotFound, obtenu %v", err)
	}
}

func TestRunParseError(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))

	_, err := e.Run("FOO name")
	if !errors.Is(err, ErrUnknownCommand) {
		t.Errorf("attendu ErrUnknownCommand, obtenu %v", err)
	}
}
