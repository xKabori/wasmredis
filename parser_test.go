package wasmredis

// =============================================================================
// TESTS DU FICHIER 1/6 — à lire juste après parser.go.
//
// Comment marchent les tests en Go :
//   - un fichier dont le nom finit par _test.go n'est compilé que par go test ;
//   - chaque fonction TestXxx(t *testing.T) est un test, et t sert à signaler
//     les échecs ("*testing.T" est un pointeur, comme *Engine) ;
//   - t.Fatalf signale l'échec ET arrête le test, ce qu'on utilise quand la
//     suite n'aurait plus de sens ; t.Errorf le signale mais continue ;
//   - on lance tout avec "go test ./..." (ou "go test -v ./..." pour le détail).
//
// Couverture de ce fichier : une forme de commande valide par test, et un test
// par cas d'erreur.
// =============================================================================

import (
	"errors"
	"testing"
)

func TestParseSet(t *testing.T) {
	cmd, err := ParseCommand("SET name matt")
	if err != nil {
		// %v affiche une valeur quelconque, ici l'erreur.
		t.Fatalf("erreur inattendue : %v", err)
	}
	// Deux structs se comparent directement avec == et !=, champ par champ.
	want := Command{Name: CmdSet, Key: "name", Value: "matt"}
	if cmd != want {
		// %+v affiche une struct AVEC le nom de ses champs, ce qui rend un
		// échec lisible : {Name:SET Key:name Value:matt Filter: TTLSeconds:0}
		t.Errorf("obtenu %+v, attendu %+v", cmd, want)
	}
}

func TestParseGet(t *testing.T) {
	cmd, err := ParseCommand("GET name")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	want := Command{Name: CmdGet, Key: "name"}
	if cmd != want {
		t.Errorf("obtenu %+v, attendu %+v", cmd, want)
	}
}

func TestParseDelete(t *testing.T) {
	cmd, err := ParseCommand("DELETE name")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	want := Command{Name: CmdDelete, Key: "name"}
	if cmd != want {
		t.Errorf("obtenu %+v, attendu %+v", cmd, want)
	}
}

func TestParseCaseInsensitive(t *testing.T) {
	cmd, err := ParseCommand("sEt name matt")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	want := Command{Name: CmdSet, Key: "name", Value: "matt"}
	if cmd != want {
		t.Errorf("obtenu %+v, attendu %+v", cmd, want)
	}
}

func TestParseValueWithSpaces(t *testing.T) {
	// Les backquotes (`) délimitent une string BRUTE : on peut y écrire des
	// guillemets sans les échapper.
	cmd, err := ParseCommand(`SET msg "hello world"`)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	want := Command{Name: CmdSet, Key: "msg", Value: "hello world"}
	if cmd != want {
		t.Errorf("obtenu %+v, attendu %+v", cmd, want)
	}
}

func TestParseExtraSpaces(t *testing.T) {
	cmd, err := ParseCommand("   GET    name   ")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	want := Command{Name: CmdGet, Key: "name"}
	if cmd != want {
		t.Errorf("obtenu %+v, attendu %+v", cmd, want)
	}
}

// --- Les cas d'erreur --------------------------------------------------------
//
// Ici, la Command renvoyée ne nous intéresse pas : on l'ignore avec "_".
// errors.Is(err, ErrX) vérifie que c'est bien CETTE erreur-là qui est remontée,
// et pas juste "une erreur quelconque".

func TestParseEmptyCommand(t *testing.T) {
	_, err := ParseCommand("   ")
	if !errors.Is(err, ErrEmptyCommand) {
		t.Errorf("attendu ErrEmptyCommand, obtenu %v", err)
	}
}

func TestParseUnknownCommand(t *testing.T) {
	_, err := ParseCommand("FOO name")
	if !errors.Is(err, ErrUnknownCommand) {
		t.Errorf("attendu ErrUnknownCommand, obtenu %v", err)
	}
}

func TestParseSetMissingValue(t *testing.T) {
	_, err := ParseCommand("SET name")
	if !errors.Is(err, ErrWrongArgs) {
		t.Errorf("attendu ErrWrongArgs, obtenu %v", err)
	}
}

func TestParseGetMissingKey(t *testing.T) {
	_, err := ParseCommand("GET")
	if !errors.Is(err, ErrWrongArgs) {
		t.Errorf("attendu ErrWrongArgs, obtenu %v", err)
	}
}

func TestParseDeleteMissingKey(t *testing.T) {
	_, err := ParseCommand("DELETE")
	if !errors.Is(err, ErrWrongArgs) {
		t.Errorf("attendu ErrWrongArgs, obtenu %v", err)
	}
}

// Sans guillemets, "hello world" fait 2 arguments de plus : la commande est
// refusée au lieu de ne garder que "hello".
func TestParseTooManyArgs(t *testing.T) {
	_, err := ParseCommand("SET msg hello world")
	if !errors.Is(err, ErrWrongArgs) {
		t.Errorf("attendu ErrWrongArgs, obtenu %v", err)
	}
}

func TestParseUnclosedQuote(t *testing.T) {
	_, err := ParseCommand(`SET msg "hello world`)
	if !errors.Is(err, ErrUnclosedQuote) {
		t.Errorf("attendu ErrUnclosedQuote, obtenu %v", err)
	}
}

// --- Le GET filtré, ajouté à l'étape 6 ---------------------------------------

func TestParseGetEquals(t *testing.T) {
	cmd, err := ParseCommand(`GET equals "hello world"`)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	// Sur un GET filtré, c'est Value qui est rempli et Key qui reste vide.
	want := Command{Name: CmdGet, Filter: FilterEquals, Value: "hello world"}
	if cmd != want {
		t.Errorf("obtenu %+v, attendu %+v", cmd, want)
	}
}

func TestParseGetContains(t *testing.T) {
	cmd, err := ParseCommand("GET CONTAINS att")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	want := Command{Name: CmdGet, Filter: FilterContains, Value: "att"}
	if cmd != want {
		t.Errorf("obtenu %+v, attendu %+v", cmd, want)
	}
}

func TestParseUnknownFilter(t *testing.T) {
	_, err := ParseCommand("GET BIGGER 30")
	if !errors.Is(err, ErrUnknownFilter) {
		t.Errorf("attendu ErrUnknownFilter, obtenu %v", err)
	}
}

// --- Le TTL, ajouté à l'étape 7 ----------------------------------------------

func TestParseSetWithTTL(t *testing.T) {
	cmd, err := ParseCommand(`SET name "matt" EX 60`)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	want := Command{Name: CmdSet, Key: "name", Value: "matt", TTLSeconds: 60}
	if cmd != want {
		t.Errorf("obtenu %+v, attendu %+v", cmd, want)
	}
}

func TestParseSetTTLNotANumber(t *testing.T) {
	_, err := ParseCommand("SET name matt EX abc")
	if !errors.Is(err, ErrInvalidTTL) {
		t.Errorf("attendu ErrInvalidTTL, obtenu %v", err)
	}
}

func TestParseSetTTLZero(t *testing.T) {
	_, err := ParseCommand("SET name matt EX 0")
	if !errors.Is(err, ErrInvalidTTL) {
		t.Errorf("attendu ErrInvalidTTL, obtenu %v", err)
	}
}

// EX sans le nombre de secondes : ce n'est plus une durée invalide, c'est un
// argument manquant.
func TestParseSetTTLMissingSeconds(t *testing.T) {
	_, err := ParseCommand("SET name matt EX")
	if !errors.Is(err, ErrWrongArgs) {
		t.Errorf("attendu ErrWrongArgs, obtenu %v", err)
	}
}
