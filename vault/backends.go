package vault

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// MemoryBackend keeps keystores in memory. Useful for tests and ephemeral use.
type MemoryBackend struct {
	mu sync.RWMutex
	m  map[string][]byte
}

// NewMemoryBackend returns an empty in-memory backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{m: map[string][]byte{}}
}

func (b *MemoryBackend) Write(id string, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	b.m[id] = cp
	return nil
}

func (b *MemoryBackend) Read(id string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	data, ok := b.m[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, nil
}

func (b *MemoryBackend) List() ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, 0, len(b.m))
	for k := range b.m {
		out = append(out, k)
	}
	return out, nil
}

func (b *MemoryBackend) Delete(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.m[id]; !ok {
		return ErrNotFound
	}
	delete(b.m, id)
	return nil
}

// FileBackend stores each keystore as a JSON file in a directory (one file per
// account), similar to a go-ethereum keystore directory.
type FileBackend struct {
	dir string
}

// NewFileBackend returns a file backend rooted at dir, creating it if needed.
func NewFileBackend(dir string) (*FileBackend, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &FileBackend{dir: dir}, nil
}

func (b *FileBackend) path(id string) string {
	return filepath.Join(b.dir, id+".json")
}

func (b *FileBackend) Write(id string, data []byte) error {
	return os.WriteFile(b.path(id), data, 0o600)
}

func (b *FileBackend) Read(id string) ([]byte, error) {
	data, err := os.ReadFile(b.path(id))
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	return data, err
}

func (b *FileBackend) List() ([]string, error) {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		out = append(out, strings.TrimSuffix(name, ".json"))
	}
	return out, nil
}

func (b *FileBackend) Delete(id string) error {
	err := os.Remove(b.path(id))
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}
