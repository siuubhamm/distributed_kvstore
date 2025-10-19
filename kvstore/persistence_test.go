package store

import (
	"os"
	"testing"
)

func TestPersistentStore(t *testing.T) {
	filename := "test_store.json"
	defer os.Remove(filename)

	ps, err := NewPersistenceStore(filename)
	if err != nil {
		t.Fatal(err)
	}

	// Set value
	ps.Set("foo", "bar")

	// Reload store → value should still exist
	ps2, _ := NewPersistenceStore(filename)
	val, err := ps2.Get("foo")
	if err != nil || val != "bar" {
		t.Errorf("expected bar, got %v", val)
	}
}
