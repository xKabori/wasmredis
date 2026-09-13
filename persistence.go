package wasmredis

// =============================================================================
// FICHIER 4/6 DANS L'ORDRE DE LECTURE — Étapes 4 et 5 : persistance et
// recovery. C'est le cœur du projet.
//
// Le problème : la RAM est volatile, mais écrire sur le disque à chaque SET
// serait beaucoup trop lent. La solution : on accumule les écritures dans une
// file en RAM (le buffer du moteur), et on vide cette file sur le disque une
// fois par seconde. Au pire, un crash fait perdre 1 seconde de données.
//
// Deux mécanismes complémentaires, comme dans le vrai Redis :
//   - l'AOF : le JOURNAL de toutes les opérations, ajoutées à la fin du fichier ;
//   - le snapshot : une PHOTO complète du state toutes les 2 minutes, suivie du
//     vidage de l'AOF (la compaction), puisque la photo rend le journal inutile.
//
// Au démarrage, Restore fait le chemin inverse : il charge la photo, puis
// rejoue le journal par-dessus.
//
// Notions Go introduites ici : JSON, goroutine, channel, ticker.
// =============================================================================

import (
	"encoding/json"
	"strings"
	"time"
)

// Operation est une écriture enregistrée dans le journal. Elle contient tout ce
// qu'il faut pour être REJOUÉE à l'identique au redémarrage.
//
// Ses champs commencent par une majuscule, et ce n'est pas un détail :
// encoding/json ne sérialise que les champs publics.
type Operation struct {
	Type  string // CmdSet ou CmdDelete
	Key   string
	Value string
	// Une date ABSOLUE, et non une durée : si on stockait "60 secondes",
	// rejouer l'AOF trois heures plus tard repousserait l'expiration d'autant.
	ExpiresAt time.Time
}

// logOperation empile une opération dans la file d'attente.
// L'appelant doit déjà tenir le verrou e.mu.
func (e *Engine) logOperation(op Operation) {
	e.buffer = append(e.buffer, op)
}

// Flush vide la file vers l'AOF. Elle est appelée une fois par seconde par
// flushLoop, et directement par les tests.
//
// Le verrou garantit aussi le "1 flush à la fois" du schéma : deux écritures
// simultanées corrompraient le fichier.
func (e *Engine) Flush() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// "Ne rien faire si la file est vide" : inutile d'ouvrir le fichier.
	if len(e.buffer) == 0 {
		return nil
	}

	// Format retenu : un objet JSON par ligne. SET name matt donne la ligne
	//   {"Type":"SET","Key":"name","Value":"matt","ExpiresAt":"0001-01-01T00:00:00Z"}
	// C'est simple à ajouter en bout de fichier, et simple à relire ligne à
	// ligne au restore.
	text := ""
	for _, op := range e.buffer {
		line, err := json.Marshal(op) // struct -> octets JSON
		if err != nil {
			return err
		}
		text += string(line) + "\n"
	}

	err := e.storage.AppendAOF([]byte(text))
	if err != nil {
		return err
	}
	// On ne vide la file QU'APRÈS une écriture réussie. En cas d'échec, les
	// opérations restent en attente et repartiront à la seconde suivante.
	e.buffer = nil
	return nil
}

// Snapshot écrit la photo complète du state, puis vide l'AOF.
func (e *Engine) Snapshot() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// json.Marshal sérialise ici toute la map d'un coup.
	data, err := json.Marshal(e.state)
	if err != nil {
		return err
	}
	err = e.storage.WriteSnapshot(data)
	if err != nil {
		return err
	}
	// La compaction : le journal des opérations passées ne sert plus, puisque
	// la photo contient déjà leur résultat.
	err = e.storage.ClearAOF()
	if err != nil {
		return err
	}
	// Même raisonnement pour les opérations encore en attente dans le buffer :
	// la photo les contient déjà.
	e.buffer = nil
	return nil
}

// Restore reconstruit la RAM depuis le disque, au démarrage.
//
// L'ORDRE est le point le plus important du fichier : la photo d'abord, le
// journal par-dessus. L'AOF contient ce qui s'est passé APRÈS la photo, donc
// l'appliquer en second, c'est rejouer l'histoire dans le bon sens.
func (e *Engine) Restore() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1) La photo.
	snapshotData, err := e.storage.ReadSnapshot()
	if err != nil {
		return err
	}
	if len(snapshotData) > 0 {
		// "var" déclare une variable avec sa valeur zéro (ici une map nil) ;
		// Unmarshal la remplira.
		var snapshot map[string]Entry
		// "&snapshot" est l'ADRESSE de la variable. Unmarshal a besoin de
		// l'adresse pour modifier la variable elle-même, sinon il ne
		// remplirait qu'une copie perdue à la sortie.
		err = json.Unmarshal(snapshotData, &snapshot)
		if err != nil {
			return err
		}
		for key, entry := range snapshot {
			// putEntry et non Set : on remplit le state et l'index SANS
			// re-logguer ces opérations, qui sont déjà sur le disque. C'est
			// aussi ce qui reconstruit l'index inversé au passage.
			e.putEntry(key, entry)
		}
	}

	// 2) Le journal, par-dessus.
	aofData, err := e.storage.ReadAOF()
	if err != nil {
		return err
	}
	// Split découpe le texte sur les retours à la ligne. Le fichier se termine
	// par un "\n", donc la dernière ligne est vide : on la saute.
	for _, line := range strings.Split(string(aofData), "\n") {
		if line == "" {
			continue // passe directement au tour suivant
		}
		var op Operation
		err = json.Unmarshal([]byte(line), &op) // octets JSON -> struct
		if err != nil {
			return err
		}
		e.applyOperation(op)
	}
	return nil
}

// applyOperation rejoue une opération sur le state et l'index, sans l'ajouter
// au buffer. C'est la différence essentielle avec Set et Delete, qui, eux,
// journalisent ce qu'ils font.
func (e *Engine) applyOperation(op Operation) {
	switch op.Type {
	case CmdSet:
		e.putEntry(op.Key, Entry{Value: op.Value, ExpiresAt: op.ExpiresAt})
	case CmdDelete:
		e.removeKey(op.Key)
	}
}

// Start lance les deux tâches de fond. "go" démarre une GOROUTINE : la fonction
// s'exécute en parallèle, et Start rend la main immédiatement.
//
// Il n'existe pas de Stop : les deux boucles tournent jusqu'à l'arrêt du
// programme. C'est volontairement hors périmètre.
func (e *Engine) Start() {
	go e.flushLoop()
	go e.snapshotLoop()
}

// flushLoop déclenche un flush toutes les secondes.
//
// ticker.C est un CHANNEL, un tuyau entre goroutines : il reçoit une valeur à
// chaque tic. "for range ticker.C" attend donc chaque tic, indéfiniment.
//
// L'erreur éventuelle est ignorée volontairement : le buffer n'ayant pas été
// vidé, la tentative suivante réessaiera d'elle-même.
func (e *Engine) flushLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		e.Flush()
	}
}

func (e *Engine) snapshotLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	for range ticker.C {
		e.Snapshot()
	}
}
