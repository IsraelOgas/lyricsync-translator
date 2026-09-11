package api

import (
	"sync"
	"testing"
)

func TestInflightSet_TryAdd(t *testing.T) {
	s := newInflightSet()

	t.Run("fresh key returns true", func(t *testing.T) {
		if !s.TryAdd("a") {
			t.Fatal("expected TryAdd on fresh key to return true")
		}
	})

	t.Run("same key returns false while in flight", func(t *testing.T) {
		if s.TryAdd("a") {
			t.Fatal("expected TryAdd on in-flight key to return false")
		}
	})

	t.Run("different key returns true", func(t *testing.T) {
		if !s.TryAdd("b") {
			t.Fatal("expected TryAdd on different key to return true")
		}
	})
}

func TestInflightSet_Remove(t *testing.T) {
	s := newInflightSet()

	if !s.TryAdd("k") {
		t.Fatal("expected first TryAdd to succeed")
	}
	s.Remove("k")
	if !s.TryAdd("k") {
		t.Fatal("expected TryAdd to succeed after Remove")
	}
}

func TestInflightSet_ConcurrentTryAdd(t *testing.T) {
	const goroutines = 50
	s := newInflightSet()

	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.TryAdd("k") {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Fatalf("expected exactly 1 successful TryAdd, got %d", successes)
	}

	// After removing, the key can be claimed again.
	s.Remove("k")
	if !s.TryAdd("k") {
		t.Fatal("expected TryAdd to succeed after Remove")
	}
}
