package wasmredis

// =============================================================================
// TESTS DU FICHIER 2/6 — à lire juste après engine.go.
//
// Deux niveaux de tests ici :
//   - les méthodes directes (Set, Get, Delete) ;
//   - le bout-à-bout avec Run, qui part d'une string brute.
//
// t.TempDir() crée un dossier temporaire, supprimé automatiquement à la fin du
// test. Le moteur a besoin d'un Storage même quand on ne teste que la RAM :
// on lui en donne un qui écrit dans ce dossier jetable.
// =============================================================================

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
	// %q affiche une string entre guillemets, ce qui rend les espaces visibles.
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

// --- Bout-à-bout : parse + exécution en une seule méthode ---------------------

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

// Run doit faire remonter l'erreur du parser sans la transformer.
func TestRunParseError(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))

	_, err := e.Run("FOO name")
	if !errors.Is(err, ErrUnknownCommand) {
		t.Errorf("attendu ErrUnknownCommand, obtenu %v", err)
	}
}
