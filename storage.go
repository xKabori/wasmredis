package wasmredis

// =============================================================================
// FICHIER 3/6 DANS L'ORDRE DE LECTURE — Étape 3 : le stockage sur le disque.
//
// Rôle : découpler le moteur du support physique. Storage est un CONTRAT (une
// interface) qui dit ce qu'on doit pouvoir faire avec un support de stockage.
// FileStorage en est une implémentation, avec deux fichiers sur le disque :
//   - appendonly.aof : le journal des opérations, écrit uniquement en ajout ;
//   - snapshot.json  : la photo complète du state.
//
// Le point important à défendre : le Storage ne manipule que des octets bruts.
// Il ne sait pas ce qu'est une opération, ni comment elle est sérialisée —
// c'est le moteur qui s'en charge (persistence.go). C'est ce découpage qui
// permettra de changer de support plus tard sans toucher au moteur.
//
// Notions Go introduites ici : interface, []byte, ouverture de fichiers, defer.
// =============================================================================

import (
	"errors"
	"os"            // accès au système de fichiers
	"path/filepath" // construire des chemins portables (\ sous Windows, / ailleurs)
)

// Une interface est une liste de méthodes. En Go, on n'écrit jamais
// "implements" : tout type qui possède ces cinq méthodes EST un Storage. Le
// compilateur le vérifie au moment où on passe un FileStorage à NewEngine.
//
// []byte est une suite d'octets bruts. On passe d'une string aux octets avec
// []byte("texte"), et des octets à une string avec string(data).
type Storage interface {
	AppendAOF(data []byte) error
	ReadAOF() ([]byte, error)
	ClearAOF() error
	WriteSnapshot(data []byte) error
	ReadSnapshot() ([]byte, error)
}

// FileStorage range les deux fichiers dans un dossier donné. Ses champs sont en
// minuscule, donc privés : personne ne peut modifier les chemins de l'extérieur.
type FileStorage struct {
	aofPath      string
	snapshotPath string
}

func NewFileStorage(dir string) *FileStorage {
	return &FileStorage{
		aofPath:      filepath.Join(dir, "appendonly.aof"),
		snapshotPath: filepath.Join(dir, "snapshot.json"),
	}
}

// AppendAOF ajoute des octets à la FIN du journal, sans jamais réécrire ce qui
// précède : c'est tout le principe d'un append only file.
func (s *FileStorage) AppendAOF(data []byte) error {
	// Les options d'ouverture se combinent avec "|" (ou binaire) :
	//   O_APPEND : écrire à la fin    O_CREATE : créer le fichier s'il manque
	//   O_WRONLY : écriture seule
	// 0644 sont les permissions Unix du fichier, ignorées en grande partie sous
	// Windows.
	//
	// On ouvre puis on referme à chaque flush, c'est-à-dire une fois par
	// seconde : le coût est négligeable, et on évite de garder un fichier
	// ouvert en permanence.
	file, err := os.OpenFile(s.aofPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	// defer repousse la fermeture à la sortie de la fonction : le fichier sera
	// fermé même si l'écriture échoue juste en dessous.
	defer file.Close()

	// Write renvoie (nombre d'octets écrits, erreur). Le nombre ne nous
	// intéresse pas, on l'ignore avec "_". Et on écrit "=" et non ":=", parce
	// que la variable err existe déjà.
	_, err = file.Write(data)
	return err
}

func (s *FileStorage) ReadAOF() ([]byte, error) {
	return readFileIfExists(s.aofPath)
}

// ClearAOF vide le journal. C'est la compaction, faite juste après un snapshot :
// la photo couvre déjà ces opérations. os.WriteFile écrase le contenu existant,
// donc y écrire une string vide revient à vider le fichier.
func (s *FileStorage) ClearAOF() error {
	return os.WriteFile(s.aofPath, []byte(""), 0644)
}

// WriteSnapshot écrase entièrement la photo précédente : seule la dernière
// compte.
func (s *FileStorage) WriteSnapshot(data []byte) error {
	return os.WriteFile(s.snapshotPath, data, 0644)
}

func (s *FileStorage) ReadSnapshot() ([]byte, error) {
	return readFileIfExists(s.snapshotPath)
}

// readFileIfExists est une fonction libre et non une méthode : elle ne dépend
// d'aucun FileStorage en particulier, et sert pour les deux fichiers.
//
// Au tout premier démarrage, les fichiers n'existent pas encore. Ce n'est pas
// une erreur : on renvoie simplement un contenu vide. errors.Is compare une
// erreur à une erreur connue, ici celle du système "le fichier n'existe pas".
func readFileIfExists(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}
