package store

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type Item struct {
	Value   string `json:"value"`
	Flags   uint32 `json:"flags"`
	Expires int64  `json:"expires"`
}

type PersistenceStore struct {
	data map[string]Item
	mu   sync.RWMutex
	file string
}

func NewPersistenceStore(filename string) (*PersistenceStore, error) {
	ps := &PersistenceStore{
		data: make(map[string]Item),
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

func (ps *PersistenceStore) Set(key string, value string, flags uint32, exptime int64) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	var expires int64
	if exptime > 0 {
		expires = time.Now().Unix() + exptime
	}

	ps.data[key] = Item{
		Value:   value,
		Flags:   flags,
		Expires: expires,
	}
	return ps.saveToFile()
}

func (ps *PersistenceStore) Get(key string) (Item, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	item, ok := ps.data[key]
	if !ok {
		return Item{}, errors.New("key not found")
	}

	if item.Expires != 0 && time.Now().Unix() > item.Expires {
		return Item{}, errors.New("key not found (expired)")
	}

	return item, nil
}

func (ps *PersistenceStore) saveToFile() error {
	bytes, err := json.MarshalIndent(ps.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ps.file, bytes, 0644)
}
