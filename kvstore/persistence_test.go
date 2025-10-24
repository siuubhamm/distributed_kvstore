package store

import (
	"os"
	"testing"
	"time"
)

func TestPersistenceStore(t *testing.T) {
	t.Run("sets, gets, and persists a value", func(t *testing.T) {
		filename := "test_store.json"
		defer os.Remove(filename)

		ps, err := NewPersistenceStore(filename)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		key := "foo"
		value := "bar"
		var flags uint32 = 123
		if err := ps.Set(key, value, flags, 0); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		ps2, err := NewPersistenceStore(filename)
		if err != nil {
			t.Fatalf("Failed to reload store: %v", err)
		}

		item, err := ps2.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if item.Value != value {
			t.Errorf("expected value %q, got %q", value, item.Value)
		}
		if item.Flags != flags {
			t.Errorf("expected flags %d, got %d", flags, item.Flags)
		}
	})

	t.Run("returns an error for expired items", func(t *testing.T) {
		filename := "test_expiry.json"
		defer os.Remove(filename)

		ps, err := NewPersistenceStore(filename)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		key := "temp"
		if err := ps.Set(key, "data", 0, 1); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		time.Sleep(2 * time.Second)

		_, err = ps.Get(key)
		if err == nil {
			t.Error("expected an error for expired key, but got none")
		}
	})

	t.Run("deletes an item", func(t *testing.T) {
		filename := "test_delete.json"
		defer os.Remove(filename)

		ps, err := NewPersistenceStore(filename)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		key := "todelete"

		ps.Set(key, "some value", 0, 0)

		_, err = ps.Get(key)
		if err == nil {
			t.Error("expected an error after deleting key, but got none")
		}
	})
}
