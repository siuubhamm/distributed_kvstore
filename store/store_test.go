package store

import (
	"testing"
)

func TestStore(t *testing.T) {
	s := NewStore()

	s.Set("foo", "bar")

	val, err := s.Get("foo")
	if err != nil || val != "bar" {
		t.Errorf("expected bar, got %v", val)
	}

	err = s.Delete("foo")
	if err != nil {
		t.Errorf("unexpected delete error: %v", err)
	}

	_, err = s.Get("foo")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
