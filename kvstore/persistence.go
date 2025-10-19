package store

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

type PersistenceStore struct {
	data map[string]string
	mu   sync.RWMutex
	file string
}

func NewPersistenceStore(filename string) (*PersistenceStore, error) {
	ps := &PersistenceStore{
		data: make(map[string]string),
		file: filename,
	}

	if _, err := os.Stat(filename); err == nil {
		bytes, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		if len(bytes) > 0 {
			if err := json.Unmarshal(bytes, &ps.data); err != nil {
				return nil, err
			}
		}
	}

	return ps, nil
}

func (ps *PersistenceStore) Set(key, value string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.data[key] = value
	return ps.saveToFile()
}

func (ps *PersistenceStore) Get(key string) (string, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	val, ok := ps.data[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return val, nil
}

func (ps *PersistenceStore) Delete(key string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	_, ok := ps.data[key]
	if !ok {
		return errors.New("key not found")
	}

	delete(ps.data, key)
	return ps.saveToFile()
}

func (ps *PersistenceStore) saveToFile() error {
	bytes, err := json.MarshalIndent(ps.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ps.file, bytes, 0644)
}
