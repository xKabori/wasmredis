package wasmredis

// =============================================================================
// FICHIER 1/6 DANS L'ORDRE DE LECTURE — Étape 1 : le parser de commandes.
//
// Rôle : transformer une ligne de texte tapée par l'utilisateur
// ("SET name matt") en une valeur Go structurée, la Command, que le moteur
// saura exécuter ensuite.
//
// Ce fichier ne stocke rien et ne touche jamais aux données : il ne fait que
// lire du texte et renvoyer soit une Command, soit une erreur. C'est pour cette
// raison qu'on y écrit des fonctions libres, et non des méthodes rattachées au
// moteur.
//
// Notions Go introduites ici : package, import, const, erreur, struct, fonction
// à plusieurs valeurs de retour, :=, switch, slice, boucle range, rune.
// =============================================================================

// Tout fichier Go commence par "package". Les fichiers .go d'un même dossier
// forment un seul package et se voient entre eux : nos fichiers n'ont donc pas
// besoin de s'importer les uns les autres.

// "import" rend utilisables des packages de la bibliothèque standard de Go.
import (
	"errors"  // fabriquer des erreurs
	"strconv" // convertir une string en nombre
	"strings" // manipuler du texte
)

// "const" déclare des valeurs fixes. On écrit CmdSet plutôt que "SET" partout :
// une faute de frappe dans un nom de constante empêche la compilation, alors
// qu'une faute dans une string ne se verrait qu'à l'exécution.
const (
	CmdSet    = "SET"
	CmdGet    = "GET"
	CmdDelete = "DELETE"
)

// Les deux filtres du GET filtré (étape 6).
const (
	FilterEquals   = "EQUALS"
	FilterContains = "CONTAINS"
)

// En Go, une erreur est une simple valeur : errors.New en fabrique une avec son
// message. On déclare ici une variable par cas d'erreur (on parle d'erreurs
// "sentinelles"), ce qui permet à l'appelant et aux tests de savoir LAQUELLE
// s'est produite, avec errors.Is(err, ErrWrongArgs).
//
// Règle de visibilité de Go : un nom qui commence par une majuscule (Command,
// ErrWrongArgs) est visible depuis l'extérieur du package, un nom en minuscule
// (splitArgs) reste privé. Il n'existe pas de mot-clé public/private.
var (
	ErrEmptyCommand   = errors.New("commande vide")
	ErrUnknownCommand = errors.New("commande inconnue (attendu : SET, GET ou DELETE)")
	ErrWrongArgs      = errors.New("nombre d'arguments incorrect (usage : SET clé valeur [EX secondes], GET clé, GET EQUALS|CONTAINS valeur, DELETE clé)")
	ErrUnknownFilter  = errors.New("filtre inconnu (attendu : EQUALS ou CONTAINS)")
	ErrInvalidTTL     = errors.New("durée EX invalide (attendu : un nombre entier de secondes supérieur à 0)")
	ErrUnclosedQuote  = errors.New("guillemet non fermé")
)

// Une struct regroupe des champs sous un nouveau type. Le type s'écrit APRÈS le
// nom du champ. Les champs inutiles à une commande donnée gardent leur "valeur
// zéro" : "" pour une string, 0 pour un int.
//
//	SET name matt        -> Name=SET     Key=name  Value=matt
//	GET name             -> Name=GET     Key=name
//	GET EQUALS matt      -> Name=GET     Filter=EQUALS  Value=matt
//	SET name matt EX 60  -> Name=SET     Key=name  Value=matt  TTLSeconds=60
//	DELETE name          -> Name=DELETE  Key=name
type Command struct {
	Name       string
	Key        string
	Value      string
	Filter     string // "" quand le GET n'est pas filtré
	TTLSeconds int    // 0 quand la clé n'expire pas
}

// ParseCommand est le point d'entrée du fichier.
//
// "(input string)" est le paramètre, "(Command, error)" sont les DEUX valeurs de
// retour : en Go, on renvoie le résultat et, en dernière position, une erreur
// éventuelle.
func ParseCommand(input string) (Command, error) {
	// ":=" déclare une variable et lui donne une valeur, le type étant deviné
	// par le compilateur. Pour modifier une variable qui existe déjà, on écrit
	// "=". TrimSpace retire les espaces et retours à la ligne aux extrémités.
	args, err := splitArgs(strings.TrimSpace(input))

	// Le réflexe de base en Go : il n'y a pas de try/catch, on teste l'erreur
	// tout de suite. "nil" veut dire "rien". Command{} est une Command vide,
	// renvoyée parce qu'il faut bien renvoyer quelque chose à la place du
	// résultat attendu.
	if err != nil {
		return Command{}, err
	}
	if len(args) == 0 { // len = longueur d'une liste
		return Command{}, ErrEmptyCommand
	}

	// args[0] est le premier élément (les index commencent à 0). ToUpper rend
	// le parser insensible à la casse : set, Set et SET donnent tous "SET".
	name := strings.ToUpper(args[0])

	// En Go, un switch ne "tombe" pas dans le cas suivant : pas besoin de break.
	// "default" attrape tout le reste, donc les commandes inconnues.
	switch name {
	case CmdSet:
		// Forme simple : SET clé valeur
		if len(args) == 3 {
			// On construit la struct en nommant les champs ; ceux qu'on omet
			// prennent leur valeur zéro.
			return Command{Name: CmdSet, Key: args[1], Value: args[2]}, nil
		}
		// Forme avec expiration (étape 7) : SET clé valeur EX secondes
		if len(args) == 5 && strings.ToUpper(args[3]) == "EX" {
			// strconv.Atoi convertit une string en int, et renvoie une erreur
			// si ce n'est pas un nombre.
			seconds, err := strconv.Atoi(args[4])
			if err != nil || seconds <= 0 {
				return Command{}, ErrInvalidTTL
			}
			return Command{Name: CmdSet, Key: args[1], Value: args[2], TTLSeconds: seconds}, nil
		}
		// Le nombre d'arguments est vérifié EXACTEMENT, et pas "au minimum" :
		// SET msg hello world (sans guillemets) est donc refusé, au lieu de
		// stocker silencieusement "hello" et de perdre le reste.
		return Command{}, ErrWrongArgs
	case CmdGet:
		// Forme simple : GET clé
		if len(args) == 2 {
			return Command{Name: CmdGet, Key: args[1]}, nil
		}
		// Forme filtrée (étape 6) : GET EQUALS valeur / GET CONTAINS texte.
		// Ici on ne cherche pas une clé, mais toutes les clés dont la valeur
		// correspond : c'est Value qui est rempli, pas Key.
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

// splitArgs découpe l'input sur les espaces, sauf entre guillemets.
// Exemple : `SET msg "hello world"` -> ["SET", "msg", "hello world"]
//
// Un simple découpage sur les espaces casserait "hello world" en deux mots. On
// lit donc l'input caractère par caractère, avec un interrupteur inQuotes qui
// indique si on se trouve à l'intérieur de guillemets.
func splitArgs(input string) ([]string, error) {
	// Un slice est une liste de taille variable. Déclaré avec "var", il démarre
	// vide (sa valeur zéro est nil), ce qui suffit pour y ajouter ensuite.
	var args []string
	current := ""
	inQuotes := false

	// "for ... range" sur une string donne à chaque tour la position et le
	// caractère. On ignore la position avec "_", car Go refuse de compiler si
	// une variable déclarée n'est pas utilisée. ch est un rune : un caractère
	// Unicode complet, donc les accents comme é sont gérés correctement.
	for _, ch := range input {
		// '"' entre apostrophes est UN caractère (un rune) ; "SET" entre
		// guillemets est une string. Ce sont deux types différents en Go.
		if ch == '"' {
			inQuotes = !inQuotes // "!" signifie "non" : on bascule l'interrupteur
		} else if ch == ' ' && !inQuotes {
			// Espace hors guillemets : le mot en cours est terminé.
			if current != "" {
				// append RENVOIE la liste agrandie : il faut réassigner args,
				// sinon l'ajout est perdu. C'est un piège classique de Go.
				args = append(args, current)
				current = ""
			}
		} else {
			// string(ch) convertit le rune en string pour pouvoir le coller au
			// mot en cours avec "+=".
			current += string(ch)
		}
	}

	// Si l'interrupteur est resté allumé, un guillemet n'a jamais été refermé.
	if inQuotes {
		return nil, ErrUnclosedQuote
	}
	// Le dernier mot n'est suivi d'aucun espace : on l'ajoute ici.
	if current != "" {
		args = append(args, current)
	}
	return args, nil
}
