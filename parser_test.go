package wasmredis

import (
	"errors"
	"testing"
)

func TestParseSet(t *testing.T) {
	cmd, err := ParseCommand("SET name matt")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	want := Command{Name: CmdSet, Key: "name", Value: "matt"}
	if cmd != want {
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

func TestParseGetEquals(t *testing.T) {
	cmd, err := ParseCommand(`GET equals "hello world"`)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
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

func TestParseSetTTLMissingSeconds(t *testing.T) {
	_, err := ParseCommand("SET name matt EX")
	if !errors.Is(err, ErrWrongArgs) {
		t.Errorf("attendu ErrWrongArgs, obtenu %v", err)
	}
}
