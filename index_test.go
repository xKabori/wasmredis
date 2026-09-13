package wasmredis

// =============================================================================
// TESTS DU FICHIER 5/6 — à lire juste après index.go.
//
// Le piège de l'index inversé n'est pas de le remplir, mais de le tenir à jour :
// après un écrasement ou une suppression, l'ancienne valeur ne doit plus
// pointer vers la clé. C'est ce que vérifient TestGetEqualsAfterOverwrite et
// TestGetEqualsAfterDelete.
//
// slices.Equal compare deux slices élément par élément (on ne peut pas utiliser
// == sur des slices en Go). Les résultats sont triés par GetEquals et
// GetContains, donc l'ordre attendu est toujours l'ordre alphabétique.
// =============================================================================

import (
	"slices"
	"testing"
)

func TestGetEquals(t *testing.T) {
	e := NewEngine(NewFileStorage(t.TempDir()))
	e.Set("alice", "30")
	e.Set("bob", "30") // deux clés peuvent porter la même valeur
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

// Après SET alice 40, l'index ne doit plus associer alice à 30.
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

	// Quand plus aucune clé ne porte la valeur, l'entrée disparaît de l'index.
	// On regarde directement le champ privé, ce qui est possible depuis un test
	// du même package.
	e.Delete("bob")
	_, found := e.index["30"]
	if found {
		t.Errorf("l'index ne doit plus contenir la valeur 30 quand plus aucune clé ne la porte")
	}
}

// contains cherche une sous-chaîne : matt et matteo correspondent, paris non.
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

// Depuis une commande texte, les clés trouvées sont renvoyées une par ligne.
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

	// En minuscules : le filtre est insensible à la casse, comme les commandes.
	value, err := e.Run("get contains ari")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if value != "city" {
		t.Errorf("obtenu %q, attendu %q", value, "city")
	}
}

// L'index n'est jamais écrit sur le disque : il est reconstruit au restore, à
// partir de la photo ET du journal. Ici alice vient du snapshot et bob de l'AOF.
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
