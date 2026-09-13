package wasmredis

// =============================================================================
// FICHIER 2/6 DANS L'ORDRE DE LECTURE — Étapes 2, 4, 6 et 7 : le cœur du moteur.
//
// Rôle : contenir les données en RAM (le "state") et exposer l'API publique du
// moteur : Set, Get, Delete, Execute (qui aiguille une Command vers la bonne
// méthode) et Run (string brute -> parse -> exécution).
//
// Ce fichier s'appuie sur trois choses définies ailleurs, qu'on peut accepter
// comme des boîtes noires pour l'instant :
//   - Entry et expireIfNeeded       -> ttl.go
//   - Operation et logOperation     -> persistence.go
//   - addToIndex et removeFromIndex -> index.go
//
// Notions Go introduites ici : map, le booléen "ok", delete, pointeur, méthode
// et receiver, mutex, defer.
// =============================================================================

import (
	"errors"
	"strings"
	"sync" // le verrou (mutex)
	"time" // les dates et les durées
)

var ErrKeyNotFound = errors.New("clé absente")

// Engine est le moteur : une struct peut regrouper des champs de types très
// différents.
type Engine struct {
	// mu protège TOUS les champs ci-dessous. Sans lui, la goroutine de flush
	// (persistence.go) et le code appelant toucheraient au state et au buffer
	// en même temps, et des accès simultanés à une map font planter Go.
	// Un sync.Mutex est utilisable tel quel, sans initialisation.
	mu sync.Mutex

	// La map est le dictionnaire clé -> valeur. C'est le store vivant, celui
	// qu'on lit en priorité.
	state map[string]Entry

	// L'index inversé (étape 6) : valeur -> ensemble des clés qui la portent.
	index map[string]map[string]bool

	// La file des écritures pas encore parties sur le disque (étape 4).
	buffer []Operation

	// Le support de stockage. C'est une interface (storage.go) : le moteur ne
	// sait pas s'il écrit dans des fichiers ou ailleurs.
	storage Storage
}

// NewEngine est le constructeur. Go n'a pas de constructeur natif : c'est une
// simple fonction, par convention nommée NewXxx.
//
// Il renvoie *Engine, un POINTEUR vers le moteur, c'est-à-dire son adresse.
// &Engine{...} crée le moteur puis renvoie son adresse. Tout le monde manipule
// ainsi le même moteur, et non une copie.
//
// make crée une map vide prête à l'emploi. Une map déclarée sans make vaut nil :
// on peut la lire, mais écrire dedans fait planter le programme.
func NewEngine(storage Storage) *Engine {
	return &Engine{
		state:   make(map[string]Entry),
		index:   make(map[string]map[string]bool),
		storage: storage,
	}
}

// Set est une MÉTHODE : une fonction rattachée à un type. "(e *Engine)" est le
// receiver, l'équivalent du "this" d'autres langages. C'est un pointeur pour
// que les modifications s'appliquent au vrai moteur et pas à une copie.
//
// Set ne prend pas le verrou : il délègue tout à SetWithTTL, qui le prend. Un
// mutex Go n'est pas réentrant : si une méthode qui tient déjà le verrou en
// appelait une autre qui le reprend, le programme se bloquerait pour toujours.
func (e *Engine) Set(key string, value string) {
	e.SetWithTTL(key, value, 0)
}

// SetWithTTL écrit une clé avec une expiration optionnelle (étape 7).
// ttlSeconds = 0 : la clé n'expire jamais.
func (e *Engine) SetWithTTL(key string, value string, ttlSeconds int) {
	e.mu.Lock()
	// "defer" repousse l'instruction à la sortie de la fonction, quel que soit
	// le chemin emprunté : on ne peut pas oublier de rendre le verrou.
	defer e.mu.Unlock()

	entry := Entry{Value: value}
	if ttlSeconds > 0 {
		// On enregistre une DATE (maintenant + la durée), et non une durée.
		// time.Duration(...) est obligatoire : Go refuse de multiplier
		// directement un int par une durée.
		entry.ExpiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	// Deux effets, toujours dans cet ordre : la RAM d'abord (le state et
	// l'index), puis la trace de l'opération dans le buffer, qui partira sur le
	// disque au prochain flush.
	e.putEntry(key, entry)
	e.logOperation(Operation{Type: CmdSet, Key: key, Value: value, ExpiresAt: entry.ExpiresAt})
}

// Get lit une clé, et renvoie une erreur si elle est absente.
func (e *Engine) Get(key string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Expiration "lazy" (étape 7) : si la clé est périmée, elle est supprimée
	// ici même, à la lecture. Il n'y a pas de goroutine de balayage.
	e.expireIfNeeded(key)

	// Lire une clé absente d'une map ne plante pas : on récupère la valeur zéro.
	// Le second résultat (ok, un booléen) dit si la clé existait vraiment.
	// C'est le seul moyen de distinguer "absente" de "présente mais vide".
	entry, ok := e.state[key]
	if !ok {
		return "", ErrKeyNotFound
	}
	return entry.Value, nil
}

// Delete supprime une clé. Supprimer une clé absente n'est pas une erreur.
func (e *Engine) Delete(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.removeKey(key)
	e.logOperation(Operation{Type: CmdDelete, Key: key})
}

// Execute est la méthode d'aiguillage : elle reçoit une Command déjà parsée et
// appelle la bonne méthode. Elle ne prend pas le verrou, car chaque méthode
// appelée le prend elle-même.
func (e *Engine) Execute(cmd Command) (string, error) {
	switch cmd.Name {
	case CmdSet:
		e.SetWithTTL(cmd.Key, cmd.Value, cmd.TTLSeconds)
		return "OK", nil
	case CmdGet:
		// GET filtré (étape 6) : on renvoie les clés trouvées, une par ligne.
		// strings.Join colle les éléments d'un slice avec un séparateur.
		if cmd.Filter == FilterEquals {
			return strings.Join(e.GetEquals(cmd.Value), "\n"), nil
		}
		if cmd.Filter == FilterContains {
			return strings.Join(e.GetContains(cmd.Value), "\n"), nil
		}
		return e.Get(cmd.Key)
	case CmdDelete:
		e.Delete(cmd.Key)
		return "OK", nil
	default:
		// Filet de sécurité : une Command fabriquée à la main avec un nom
		// inconnu n'est pas passée par le parser.
		return "", ErrUnknownCommand
	}
}

// Run est le bout-à-bout demandé : string brute -> parse -> exécution.
// Une erreur de parsing est simplement propagée à l'appelant.
func (e *Engine) Run(input string) (string, error) {
	cmd, err := ParseCommand(input)
	if err != nil {
		return "", err
	}
	return e.Execute(cmd)
}

// --- Méthodes privées --------------------------------------------------------
//
// putEntry et removeKey modifient le state ET l'index, sans rien écrire dans le
// buffer. C'est exactement ce qu'il faut au restore (persistence.go), qui
// rejoue des opérations déjà présentes sur le disque et ne doit surtout pas les
// re-logguer.
//
// Elles commencent par une minuscule : elles sont privées au package. Elles ne
// prennent pas le verrou, car l'appelant le tient déjà.

func (e *Engine) putEntry(key string, entry Entry) {
	// On retire d'abord l'ancienne version de la clé, sinon l'index la
	// garderait sous son ancienne valeur (SET alice 30 puis SET alice 40).
	e.removeKey(key)
	e.state[key] = entry
	e.addToIndex(entry.Value, key)
}

func (e *Engine) removeKey(key string) {
	oldEntry, ok := e.state[key]
	if !ok {
		return // rien à supprimer
	}
	// delete est une fonction intégrée au langage, qui retire une clé d'une map.
	delete(e.state, key)
	e.removeFromIndex(oldEntry.Value, key)
}
