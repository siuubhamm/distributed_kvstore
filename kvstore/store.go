package store

import (
	"errors"
)

type Store struct {
	data map[string]string
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) Set(key, value string) {
	s.data[key] = value
}

func (s *Store) Get(key string) (string, error) {
	value, ok := s.data[key]

	if ok {
		return value, nil
	}

	return "", errors.New("key not found")
}

// func (s *Store) Delete(key string) error {
// 	_, ok := s.data[key]

// 	if ok {
// 		delete(s.data, key)
// 		return nil
// 	}

// 	return errors.New("key not found")
// }
