package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New(10)

	// Set and retrieve a value
	c.Set("key1", "value1", 1*time.Hour)
	val, ok := c.Get("key1")
	if !ok {
		t.Error("Expected key1 to be found")
	}
	if val != "value1" {
		t.Errorf("Expected value 'value1', got '%v'", val)
	}
}

func TestCache_GetNonExistent(t *testing.T) {
	c := New(10)

	val, ok := c.Get("nonexistent")
	if ok {
		t.Error("Expected key to not be found")
	}
	if val != nil {
		t.Errorf("Expected nil value, got %v", val)
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	c := New(10)

	// Set with short TTL
	c.Set("expire", "value", 50*time.Millisecond)

	// Should exist immediately
	val, ok := c.Get("expire")
	if !ok {
		t.Error("Expected key to exist immediately after set")
	}
	if val != "value" {
		t.Errorf("Expected value 'value', got '%v'", val)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	val, ok = c.Get("expire")
	if ok {
		t.Error("Expected key to be expired")
	}
	if val != nil {
		t.Errorf("Expected nil after expiration, got %v", val)
	}
}

func TestCache_TTLExpiration_MultipleItems(t *testing.T) {
	c := New(10)

	// Set multiple items with different TTLs
	c.Set("short", "value1", 50*time.Millisecond)
	c.Set("long", "value2", 5*time.Second)

	// Wait for short TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Short should be expired
	_, ok := c.Get("short")
	if ok {
		t.Error("Expected 'short' key to be expired")
	}

	// Long should still exist
	val, ok := c.Get("long")
	if !ok {
		t.Error("Expected 'long' key to still exist")
	}
	if val != "value2" {
		t.Errorf("Expected 'value2', got '%v'", val)
	}
}

func TestCache_SizeLimit(t *testing.T) {
	c := New(3)

	// Fill cache to capacity
	c.Set("key1", "value1", 1*time.Hour)
	c.Set("key2", "value2", 1*time.Hour)
	c.Set("key3", "value3", 1*time.Hour)

	// All should exist
	if _, ok := c.Get("key1"); !ok {
		t.Error("Expected key1 to exist")
	}
	if _, ok := c.Get("key2"); !ok {
		t.Error("Expected key2 to exist")
	}
	if _, ok := c.Get("key3"); !ok {
		t.Error("Expected key3 to exist")
	}

	// Add one more - should evict one
	c.Set("key4", "value4", 1*time.Hour)

	// key4 should exist
	if _, ok := c.Get("key4"); !ok {
		t.Error("Expected key4 to exist after eviction")
	}

	// Should have exactly 3 items (one was evicted)
	count := 0
	keys := []string{"key1", "key2", "key3", "key4"}
	for _, key := range keys {
		if _, ok := c.Get(key); ok {
			count++
		}
	}
	if count != 3 {
		t.Errorf("Expected exactly 3 items in cache, got %d", count)
	}
}

func TestCache_UpdateExistingKey(t *testing.T) {
	c := New(10)

	// Set initial value
	c.Set("key", "value1", 1*time.Hour)

	// Update with new value
	c.Set("key", "value2", 1*time.Hour)

	// Should get updated value
	val, ok := c.Get("key")
	if !ok {
		t.Error("Expected key to exist")
	}
	if val != "value2" {
		t.Errorf("Expected updated value 'value2', got '%v'", val)
	}
}

func TestCache_UpdateWithNewTTL(t *testing.T) {
	c := New(10)

	// Set with short TTL
	c.Set("key", "value1", 50*time.Millisecond)

	// Wait a bit but not enough to expire
	time.Sleep(30 * time.Millisecond)

	// Update with long TTL
	c.Set("key", "value2", 5*time.Second)

	// Wait for original TTL to pass
	time.Sleep(50 * time.Millisecond)

	// Should still exist with new TTL
	val, ok := c.Get("key")
	if !ok {
		t.Error("Expected key to exist with new TTL")
	}
	if val != "value2" {
		t.Errorf("Expected 'value2', got '%v'", val)
	}
}

func TestCache_DifferentValueTypes(t *testing.T) {
	c := New(10)

	// Test different types
	c.Set("string", "text", 1*time.Hour)
	c.Set("int", 42, 1*time.Hour)
	c.Set("bool", true, 1*time.Hour)
	c.Set("struct", struct{ Name string }{"test"}, 1*time.Hour)

	// Retrieve and verify
	if val, ok := c.Get("string"); !ok || val != "text" {
		t.Error("String value mismatch")
	}
	if val, ok := c.Get("int"); !ok || val != 42 {
		t.Error("Int value mismatch")
	}
	if val, ok := c.Get("bool"); !ok || val != true {
		t.Error("Bool value mismatch")
	}
	if val, ok := c.Get("struct"); !ok {
		t.Error("Struct not found")
	} else {
		s, ok := val.(struct{ Name string })
		if !ok || s.Name != "test" {
			t.Error("Struct value mismatch")
		}
	}
}

func TestCache_ZeroTTL(t *testing.T) {
	c := New(10)

	// Set with zero TTL (immediate expiration)
	c.Set("zero", "value", 0)

	// Should be immediately expired
	_, ok := c.Get("zero")
	if ok {
		t.Error("Expected key with zero TTL to be expired immediately")
	}
}

func TestCache_NegativeTTL(t *testing.T) {
	c := New(10)

	// Set with negative TTL (already expired)
	c.Set("negative", "value", -1*time.Hour)

	// Should be expired
	_, ok := c.Get("negative")
	if ok {
		t.Error("Expected key with negative TTL to be expired")
	}
}

func TestCache_Concurrency(t *testing.T) {
	c := New(100)
	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune('a' + (id*numOperations+j)%26))
				c.Set(key, id*numOperations+j, 100*time.Millisecond)
			}
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune('a' + (id*numOperations+j)%26))
				c.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// No assertion needed - test passes if no race conditions
}

func TestCache_ConcurrentExpiration(t *testing.T) {
	c := New(50)

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			c.Set("key", i, 10*time.Millisecond)
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Reader goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			c.Get("key")
			time.Sleep(2 * time.Millisecond)
		}
	}()

	wg.Wait()
}

func TestCache_ZeroSize(t *testing.T) {
	c := New(0)

	// Should not panic
	c.Set("key", "value", 1*time.Hour)

	// Every set should evict the previous item
	c.Set("key2", "value2", 1*time.Hour)

	// At most one item should exist
	count := 0
	if _, ok := c.Get("key"); ok {
		count++
	}
	if _, ok := c.Get("key2"); ok {
		count++
	}
	if count > 1 {
		t.Errorf("Expected at most 1 item with size 0, got %d", count)
	}
}

func TestCache_LargeDataset(t *testing.T) {
	c := New(1000)

	// Add many items
	for i := 0; i < 1000; i++ {
		c.Set(string(rune(i)), i, 1*time.Hour)
	}

	// Verify some items exist
	retrieved := 0
	for i := 0; i < 1000; i++ {
		if _, ok := c.Get(string(rune(i))); ok {
			retrieved++
		}
	}

	// Should have exactly 1000 items (cache size)
	if retrieved != 1000 {
		t.Errorf("Expected 1000 items in cache, got %d", retrieved)
	}

	// Add one more - should evict one
	c.Set("extra", "value", 1*time.Hour)
	if _, ok := c.Get("extra"); !ok {
		t.Error("Expected 'extra' to exist after eviction")
	}

	// Count again
	retrieved = 0
	for i := 0; i < 1000; i++ {
		if _, ok := c.Get(string(rune(i))); ok {
			retrieved++
		}
	}
	if _, ok := c.Get("extra"); ok {
		retrieved++
	}

	// Should still have exactly 1000 items
	if retrieved != 1000 {
		t.Errorf("Expected exactly 1000 items after eviction, got %d", retrieved)
	}
}

func BenchmarkCache_Set(b *testing.B) {
	c := New(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set("key", i, 1*time.Hour)
	}
}

func BenchmarkCache_Get(b *testing.B) {
	c := New(1000)
	c.Set("key", "value", 1*time.Hour)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("key")
	}
}

func BenchmarkCache_SetParallel(b *testing.B) {
	c := New(1000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set("key", i, 1*time.Hour)
			i++
		}
	})
}

func BenchmarkCache_GetParallel(b *testing.B) {
	c := New(1000)
	c.Set("key", "value", 1*time.Hour)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Get("key")
		}
	})
}
