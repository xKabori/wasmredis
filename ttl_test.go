package wasmredis

// =============================================================================
// TESTS DU FICHIER 6/6 — à lire juste après ttl.go.
//
// Pour tester une expiration, il faut que le temps passe : on écrit donc avec
// EX 1 (une seconde) puis on attend 1,1 seconde avec time.Sleep. C'est ce qui
// explique que la suite de tests dure quelques secondes.
//
// L'alternative propre serait d'injecter une fausse horloge dans le moteur,
// mais ça complique le code pour un gain limité ici : on reste sur l'attente
// réelle, avec une durée courte.
// =============================================================================

import (
	"errors"
	"slices"
	"testing"
	"time"
)

// Avec un TTL de 60 secondes, la clé est évidemment toujours là.
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

// Le cœur de l'expiration lazy : après l'échéance, la lecture doit répondre
// "clé absente" ET faire réellement le ménage dans le state.
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

// Un GET filtré est une lecture lui aussi : il ne doit pas renvoyer de clés
// périmées. alice et carol expirent, bob et dave restent.
func TestTTLExpiredFilters(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.SetWithTTL("alice", "30", 1)
	e.Set("bob", "30")
	e.SetWithTTL("carol", "paris", 1)
	e.Set("dave", "paris")

	time.Sleep(1100 * time.Millisecond)

	got := e.GetEquals("30") // passe par l'index
	want := []string{"bob"}
	if !slices.Equal(got, want) {
		t.Errorf("equals : obtenu %v, attendu %v", got, want)
	}

	got = e.GetContains("par") // passe par le parcours complet
	want = []string{"dave"}
	if !slices.Equal(got, want) {
		t.Errorf("contains : obtenu %v, attendu %v", got, want)
	}
}

// La date d'expiration doit survivre au redémarrage, par le snapshot comme par
// le journal. C'est ce qui prouve qu'on stocke une date absolue : si on
// stockait une durée, le restore la ferait repartir de zéro.
func TestTTLKeptAfterRestore(t *testing.T) {
	dir := t.TempDir()

	e1 := NewEngine(NewFileStorage(dir))
	e1.SetWithTTL("session", "abc", 60)
	err := e1.Snapshot() // session part dans la photo
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	e1.SetWithTTL("token", "xyz", 120)
	err = e1.Flush() // token part dans le journal
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
