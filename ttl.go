package wasmredis

// =============================================================================
// FICHIER 6/6 DANS L'ORDRE DE LECTURE — Étape 7 : le TTL (time to live), en
// expiration "lazy".
//
// Une entrée peut porter une date d'expiration. Rien ne surveille l'horloge en
// arrière-plan : c'est à la LECTURE qu'on regarde si l'entrée est périmée, et
// si c'est le cas on la supprime et on la traite comme absente. C'est ce que
// veut dire "lazy". Le vrai Redis combine ça avec un balayage actif, qui est
// hors périmètre ici.
//
// Notions Go introduites ici : time.Time, valeur zéro d'une date, méthode avec
// receiver par valeur.
// =============================================================================

import "time"

// Entry est ce qu'on range dans le state : la valeur, plus une date
// d'expiration facultative.
//
// La valeur zéro d'un time.Time (une date vide) sert de "pas d'expiration".
// C'est plus simple qu'un booléen supplémentaire, et c'est exactement ce que
// contient une Entry créée par un SET sans EX.
type Entry struct {
	Value     string
	ExpiresAt time.Time
}

// isExpired dit si l'entrée est périmée.
//
// Le receiver "(entry Entry)" est une VALEUR et non un pointeur : la méthode
// travaille sur une copie de l'entrée. C'est suffisant ici, puisqu'elle ne fait
// que lire. On met un pointeur (comme *Engine ailleurs) quand la méthode doit
// modifier ce sur quoi elle est appelée.
func (entry Entry) isExpired() bool {
	if entry.ExpiresAt.IsZero() { // date vide = pas d'expiration
		return false
	}
	return time.Now().After(entry.ExpiresAt)
}

// expireIfNeeded est le mécanisme d'expiration lazy. Il est appelé par Get
// (engine.go) et par les deux filtres (index.go), qui sont aussi des lectures.
// Il renvoie true si la clé était expirée.
//
// L'appelant doit déjà tenir le verrou e.mu.
func (e *Engine) expireIfNeeded(key string) bool {
	entry, ok := e.state[key]
	// "||" est un "ou" court-circuité : si !ok est vrai, isExpired n'est même
	// pas appelé.
	if !ok || !entry.isExpired() {
		return false
	}
	// Une expiration est traitée comme une suppression ordinaire : on retire la
	// clé du state et de l'index, et on journalise un DELETE pour que le disque
	// reste cohérent avec la RAM.
	e.removeKey(key)
	e.logOperation(Operation{Type: CmdDelete, Key: key})
	return true
}
