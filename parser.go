package wasmredis

import (
	"errors"
	"strconv"
	"strings"
)

const (
	CmdSet    = "SET"
	CmdGet    = "GET"
	CmdDelete = "DELETE"
)

const (
	FilterEquals   = "EQUALS"
	FilterContains = "CONTAINS"
)

var (
	ErrEmptyCommand   = errors.New("commande vide")
	ErrUnknownCommand = errors.New("commande inconnue (attendu : SET, GET ou DELETE)")
	ErrWrongArgs      = errors.New("nombre d'arguments incorrect (usage : SET clé valeur [EX secondes], GET clé, GET EQUALS|CONTAINS valeur, DELETE clé)")
	ErrUnknownFilter  = errors.New("filtre inconnu (attendu : EQUALS ou CONTAINS)")
	ErrInvalidTTL     = errors.New("durée EX invalide (attendu : un nombre entier de secondes supérieur à 0)")
	ErrUnclosedQuote  = errors.New("guillemet non fermé")
)

type Command struct {
	Name       string
	Key        string
	Value      string
	Filter     string
	TTLSeconds int
}

func ParseCommand(input string) (Command, error) {
	args, err := splitArgs(strings.TrimSpace(input))

	if err != nil {
		return Command{}, err
	}
	if len(args) == 0 {
		return Command{}, ErrEmptyCommand
	}

	name := strings.ToUpper(args[0])

	switch name {
	case CmdSet:
		if len(args) == 3 {
			return Command{Name: CmdSet, Key: args[1], Value: args[2]}, nil
		}
		if len(args) == 5 && strings.ToUpper(args[3]) == "EX" {
			seconds, err := strconv.Atoi(args[4])
			if err != nil || seconds <= 0 {
				return Command{}, ErrInvalidTTL
			}
			return Command{Name: CmdSet, Key: args[1], Value: args[2], TTLSeconds: seconds}, nil
		}
		return Command{}, ErrWrongArgs
	case CmdGet:
		if len(args) == 2 {
			return Command{Name: CmdGet, Key: args[1]}, nil
		}
		if len(args) == 3 {
			filter := strings.ToUpper(args[1])
			if filter != FilterEquals && filter != FilterContains {
				return Command{}, ErrUnknownFilter
			}
			return Command{Name: CmdGet, Filter: filter, Value: args[2]}, nil
		}
		return Command{}, ErrWrongArgs
	case CmdDelete:
		if len(args) != 2 {
			return Command{}, ErrWrongArgs
		}
		return Command{Name: CmdDelete, Key: args[1]}, nil
	default:
		return Command{}, ErrUnknownCommand
	}
}

func splitArgs(input string) ([]string, error) {
	var args []string
	current := ""
	inQuotes := false

	for _, ch := range input {
		if ch == '"' {
			inQuotes = !inQuotes
		} else if ch == ' ' && !inQuotes {
			if current != "" {
				args = append(args, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if inQuotes {
		return nil, ErrUnclosedQuote
	}
	if current != "" {
		args = append(args, current)
	}
	return args, nil
}
