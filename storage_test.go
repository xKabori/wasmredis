package wasmredis

import "testing"

func TestFileStorageMissingFiles(t *testing.T) {
	s := NewFileStorage(t.TempDir())

	aof, err := s.ReadAOF()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if len(aof) != 0 {
		t.Errorf("AOF attendu vide, obtenu %q", aof)
	}

	snapshot, err := s.ReadSnapshot()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if len(snapshot) != 0 {
		t.Errorf("snapshot attendu vide, obtenu %q", snapshot)
	}
}

func TestFileStorageAppendAOF(t *testing.T) {
	s := NewFileStorage(t.TempDir())

	err := s.AppendAOF([]byte("ligne 1\n"))
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	err = s.AppendAOF([]byte("ligne 2\n"))
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	aof, err := s.ReadAOF()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if string(aof) != "ligne 1\nligne 2\n" {
		t.Errorf("contenu AOF incorrect : %q", aof)
	}
}

func TestFileStorageClearAOF(t *testing.T) {
	s := NewFileStorage(t.TempDir())

	err := s.AppendAOF([]byte("ligne 1\n"))
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	err = s.ClearAOF()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	aof, err := s.ReadAOF()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if len(aof) != 0 {
		t.Errorf("AOF attendu vide, obtenu %q", aof)
	}
}

func TestFileStorageSnapshot(t *testing.T) {
	s := NewFileStorage(t.TempDir())

	err := s.WriteSnapshot([]byte("premier"))
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	err = s.WriteSnapshot([]byte("second"))
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	snapshot, err := s.ReadSnapshot()
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if string(snapshot) != "second" {
		t.Errorf("snapshot incorrect : %q", snapshot)
	}
}
