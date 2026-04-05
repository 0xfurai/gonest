package cache

import (
	"testing"
	"time"
)

func TestMemoryStore_SetAndGet(t *testing.T) {
	store := NewMemoryStore()
	store.Set("key1", "value1", 0)

	val, ok := store.Get("key1")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got %v", val)
	}
}

func TestMemoryStore_GetMissing(t *testing.T) {
	store := NewMemoryStore()
	_, ok := store.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestMemoryStore_TTL_Expired(t *testing.T) {
	store := NewMemoryStore()
	store.Set("key1", "value1", 50*time.Millisecond)

	// Should exist immediately
	val, ok := store.Get("key1")
	if !ok || val != "value1" {
		t.Error("expected key to exist before TTL")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	_, ok = store.Get("key1")
	if ok {
		t.Error("expected key to be expired")
	}
}

func TestMemoryStore_TTL_NotExpired(t *testing.T) {
	store := NewMemoryStore()
	store.Set("key1", "value1", 5*time.Second)

	val, ok := store.Get("key1")
	if !ok || val != "value1" {
		t.Error("expected key to exist before TTL")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	store.Set("key1", "value1", 0)
	store.Delete("key1")

	_, ok := store.Get("key1")
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	store := NewMemoryStore()
	store.Set("key1", "value1", 0)
	store.Set("key2", "value2", 0)
	store.Clear()

	_, ok1 := store.Get("key1")
	_, ok2 := store.Get("key2")
	if ok1 || ok2 {
		t.Error("expected all keys to be cleared")
	}
}

func TestMemoryStore_Overwrite(t *testing.T) {
	store := NewMemoryStore()
	store.Set("key1", "value1", 0)
	store.Set("key1", "value2", 0)

	val, _ := store.Get("key1")
	if val != "value2" {
		t.Errorf("expected 'value2', got %v", val)
	}
}

func TestMemoryStore_NoTTL(t *testing.T) {
	store := NewMemoryStore()
	store.Set("persistent", "data", 0)

	time.Sleep(10 * time.Millisecond)

	val, ok := store.Get("persistent")
	if !ok || val != "data" {
		t.Error("expected persistent key to exist without TTL")
	}
}
