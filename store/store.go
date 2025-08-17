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

