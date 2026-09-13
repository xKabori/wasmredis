package wasmredis

// =============================================================================
// FICHIER 5/6 DANS L'ORDRE DE LECTURE — Étape 6 : l'index inversé et le GET
// filtré.
//
// Le state répond à la question "quelle valeur porte la clé name ?". L'index
// inversé répond à la question inverse : "quelles clés portent la valeur 30 ?".
// C'est une map valeur -> ensemble de clés, tenue à jour à chaque écriture et à
// chaque suppression, et reconstruite au restore.
//
// Résultat : GET EQUALS est un accès direct à une case de map, au lieu de
// parcourir toute la base. GET CONTAINS, lui, cherche une sous-chaîne : aucun
// index simple ne sait faire ça, donc c'est un parcours complet assumé.
//
// Notions Go introduites ici : l'ensemble (map[string]bool), la map imbriquée,
// l'ordre aléatoire des maps, le tri.
// =============================================================================

import (
	"sort"
	"strings"
)

// Un "ensemble" s'écrit map[string]bool en Go : seules les clés de la map
// comptent, le true n'est qu'un remplissage. L'index est donc une map de maps :
// index["30"]["alice"] = true.
//
// L'appelant doit déjà tenir le verrou e.mu.
func (e *Engine) addToIndex(value string, key string) {
	// La sous-map n'existe pas au premier ajout : lire une case absente donne
	// nil, et écrire dans une map nil fait planter le programme. On la crée
	// donc avant.
	if e.index[value] == nil {
		e.index[value] = make(map[string]bool)
	}
	e.index[value][key] = true
}

// removeFromIndex retire une clé de l'ensemble, et supprime l'ensemble lui-même
// dès qu'il devient vide : l'index ne garde pas de valeurs fantômes.
//
// L'appelant doit déjà tenir le verrou e.mu.
func (e *Engine) removeFromIndex(value string, key string) {
	delete(e.index[value], key)
	if len(e.index[value]) == 0 {
		delete(e.index, value)
	}
}

// GetEquals renvoie les clés dont la valeur est exactement celle demandée.
// C'est un accès direct à l'index, sans parcourir le state.
func (e *Engine) GetEquals(value string) []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	var keys []string
	// "for key := range maMap" parcourt les CLÉS de la map.
	for key := range e.index[value] {
		// Un filtre est une lecture : l'expiration lazy s'applique donc ici
		// aussi (ttl.go). Go autorise à supprimer une entrée d'une map pendant
		// qu'on la parcourt.
		if !e.expireIfNeeded(key) {
			keys = append(keys, key)
		}
	}
	// Go parcourt les maps dans un ordre ALÉATOIRE, et c'est volontaire dans le
	// langage. Sans tri, deux appels identiques renverraient les clés dans un
	// ordre différent et les tests seraient instables.
	sort.Strings(keys)
	return keys
}

// GetContains renvoie les clés dont la valeur contient la sous-chaîne demandée.
// Ici aucun index ne peut aider : on parcourt tout le state.
func (e *Engine) GetContains(substring string) []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	var keys []string
	// Avec deux variables, "range" sur une map donne la clé ET la valeur.
	for key, entry := range e.state {
		if e.expireIfNeeded(key) {
			continue
		}
		if strings.Contains(entry.Value, substring) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
