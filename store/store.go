package kvstore

import (
	"errors"
)

// In memory key value store.
// The string: string type helps in adapting to any type of data.
type Store struct {
	data map[string]string
}

// To initialize a new Store
func NewStore() *Store {
	return &Store{
		data : make(map[string][string])
	}
}

// Insert OR Update a value.
func (s *Store) Set(key, value string) {
	s.data[key]
}

// Get the value for a key or return an error.
func (s * Store) Get(key string) (string, error) {
	value, ok = s.data[key]

	if okay {
		return value, nil
	}

	return "", errors.New("Key not found")
}

// Delete a key from the store.
func (s *Store) (key string) error {
	value, ok = s.data[key]

	if ok {
		delete(s.data, key)
		return nil
	}

	return errors.New("Key not found")
}