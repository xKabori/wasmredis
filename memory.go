package wasmredis

type MemoryStorage struct {
	aof      []byte
	snapshot []byte
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{}
}

func (s *MemoryStorage) AppendAOF(data []byte) error {
	s.aof = append(s.aof, data...)
	return nil
}

func (s *MemoryStorage) ReadAOF() ([]byte, error) {
	return s.aof, nil
}

func (s *MemoryStorage) ClearAOF() error {
	s.aof = nil
	return nil
}

func (s *MemoryStorage) WriteSnapshot(data []byte) error {
	s.snapshot = data
	return nil
}

func (s *MemoryStorage) ReadSnapshot() ([]byte, error) {
	return s.snapshot, nil
}
