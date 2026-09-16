package wasmredis

import (
	"errors"
	"os"
	"path/filepath"
)

type Storage interface {
	AppendAOF(data []byte) error
	ReadAOF() ([]byte, error)
	ClearAOF() error
	WriteSnapshot(data []byte) error
	ReadSnapshot() ([]byte, error)
}

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

func (s *FileStorage) AppendAOF(data []byte) error {
	file, err := os.OpenFile(s.aofPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

func (s *FileStorage) ReadAOF() ([]byte, error) {
	return readFileIfExists(s.aofPath)
}

func (s *FileStorage) ClearAOF() error {
	return os.WriteFile(s.aofPath, []byte(""), 0644)
}

func (s *FileStorage) WriteSnapshot(data []byte) error {
	return os.WriteFile(s.snapshotPath, data, 0644)
}

func (s *FileStorage) ReadSnapshot() ([]byte, error) {
	return readFileIfExists(s.snapshotPath)
}

func readFileIfExists(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}
