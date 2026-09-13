package wasmredis

// =============================================================================
// TESTS DU FICHIER 4/6 — à lire juste après persistence.go. Ce sont les tests
// les plus importants du projet.
//
// LE TEST EN OR est TestRestoreFromAOF : on écrit des données, on flush, on
// crée un NOUVEAU moteur sur le MÊME dossier, on restaure, et on vérifie que
// l'état est identique. C'est la démonstration que les données survivent à un
// redémarrage.
//
// Les tests appellent Flush et Snapshot directement, au lieu d'attendre les
// tickers : c'est instantané et déterministe. Un seul test
// (TestStartFlushesInBackground) vérifie le déclenchement automatique, et lui
// doit forcément patienter une seconde.
//
// Ces tests sont dans le même package que le code, donc ils peuvent regarder
// les champs privés (e.buffer, e.state). C'est très pratique pour vérifier des
// choses invisibles de l'extérieur, comme "le buffer a bien été vidé".
// =============================================================================

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

	// Deux opérations écrites, donc deux lignes JSON dans le journal.
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

// "Ne rien faire si la file est vide" : le second flush ne doit pas réécrire
// l'opération déjà partie sur le disque.
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

// Le snapshot photographie tout le state (y compris ce qui était encore dans le
// buffer), puis vide le journal : c'est la compaction.
func TestSnapshotWritesStateAndClearsAOF(t *testing.T) {
	storage := NewFileStorage(t.TempDir())
	e := NewEngine(storage)
	e.Set("name", "matt")
	err := e.Flush()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	e.Set("age", "30") // pas encore flushé au moment du snapshot

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

// Le seul test qui vérifie le ticker : on lance les tâches de fond, on écrit,
// et on attend un peu plus d'une seconde que le flush automatique se déclenche.
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

// LE TEST EN OR : écrire, flush, "tuer" le moteur, en relancer un neuf sur le
// même dossier, restaurer, et retrouver exactement le même état.
func TestRestoreFromAOF(t *testing.T) {
	dir := t.TempDir()

	e1 := NewEngine(NewFileStorage(dir))
	e1.Set("name", "matt")
	e1.Set("age", "30")
	e1.Set("temp", "x")
	e1.Delete("temp") // le DELETE doit être rejoué lui aussi
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

// La même chose avec les DEUX mécanismes : la photo d'abord, le journal
// par-dessus. age passe de 30 (photo) à 31 (journal) et name est supprimé après
// la photo : si l'ordre était inversé, ce test échouerait.
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

// Tout premier démarrage : le dossier est vide, ce n'est pas une erreur.
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

// Exigence facile à oublier : rejouer le journal ne doit PAS remplir le buffer,
// sinon le premier flush réécrirait sur le disque tout ce qu'on vient d'y lire.
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

// assertSameState compare deux states, clé par clé. C'est un helper partagé par
// plusieurs tests, y compris ceux du TTL.
//
// t.Helper() indique à Go que cette fonction est un utilitaire : en cas
// d'échec, la ligne affichée est celle du test appelant, pas celle d'ici.
//
// On compare les dates avec .Equal et non avec == : après un aller-retour en
// JSON, deux dates représentant le même instant peuvent différer par des
// détails internes, et == les verrait comme différentes.
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
