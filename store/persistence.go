package store

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

// Persistence is key-value store which is
// backed by a json file.
type PersistenceStore struct {
	data map[string]string
	mu   sync.RWMutex
	file string
}

// NewPersistanceStore creates a new PersistenceStore,
// loading data if file exists.
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

// Set stores a key-value pair and saves it to file.
func (ps *PersistenceStore) Set(key, value string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.data[key] = value
	return ps.saveToFile()
}

// Get retrieves a value.
func (ps *PersistenceStore) Get(key string) (string, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	val, ok := ps.data[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return val, nil
}

// Delete removes a key and saves it to a file.
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

// saveToFile writes the current map into the json file.
func (ps *PersistenceStore) saveToFile() error {
	bytes, err := json.MarshalIndent(ps.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ps.file, bytes, 0644)
}
