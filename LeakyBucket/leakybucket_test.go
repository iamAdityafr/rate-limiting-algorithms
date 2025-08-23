package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestAllow(t *testing.T) {
	lb, err := NewLeakyBucket(5, 1.0, PerSecond)
	if err != nil {
		t.Fatalf("NewLeakyBucket could'nt initialise: %v", err)
	}

	for i := range 5 {
		if !lb.Allow(1) {
			t.Errorf("Allow(1) at %d should succeed", i)
		}
	}

	if lb.Allow(1) {
		t.Error("Allow(1) after full should fail")
	}

	time.Sleep(3100 * time.Millisecond)

	for i := range 3 {
		if !lb.Allow(1) {
			t.Errorf("Allow(1) after %d leak should succeed", i)
		}
	}
}

// blocks and waits until space is available
func TestTake(t *testing.T) {
	lb, err := NewLeakyBucket(3, 2.0, PerSecond)
	if err != nil {
		t.Fatalf("NewLeakyBucket couldnt initialise: %v", err)
	}

	for range 3 {
		lb.Allow(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err = lb.Take(ctx, 2)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Take failed: %v", err)
	}

	// take at least ~1 second to free up space for 2 requests
	if elapsed < 900*time.Millisecond {
		t.Errorf("should have waited ~1s but waited only %v", elapsed)
	}
}

func TestTake_ContextCancelled(t *testing.T) {
	lb, err := NewLeakyBucket(1, 1.0, PerSecond)
	if err != nil {
		t.Fatalf("NewLeakyBucket failed: %v", err)
	}

	lb.Allow(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	err = lb.Take(ctx, 1)
	if err == nil {
		t.Error("Should return error when context is cancelled")
	}
	if err != context.Canceled {
		t.Errorf("Should return context.Canceled but returned: %v", err)
	}
}

func TestStats(t *testing.T) {
	lb, err := NewLeakyBucket(3, 1.0, PerSecond)
	if err != nil {
		t.Fatalf("NewLeakyBucket failed: %v", err)
	}

	lb.Allow(2)
	lb.Allow(2) // drops 2, queues 0

	processed, dropped, queue := lb.Stats()
	if dropped != 2 {
		t.Errorf("Expected 2 dropped, got %d", dropped)
	}
	if queue < 1.9 || queue > 2.1 {
		t.Errorf("Expected queue≈2, got %.2f", queue)
	}

	// should process ~2 requests
	time.Sleep(2100 * time.Millisecond)

	processed, dropped, queue = lb.Stats()

	// processed some requests
	if processed == 0 && dropped == 0 {
		t.Errorf("Processing expected, got processed=%d, dropped=%d", processed, dropped)
	}

	// Queue would get smaller after processing some
	if queue >= 2.0 {
		t.Errorf("Queue shouldve decreased, got %.2f", queue)
	}
}

// tests for bucket is safe under concurrent access (Concurrency Test)
func TestConcurrency(t *testing.T) {
	lb, err := NewLeakyBucket(100, 10.0, PerSecond)
	if err != nil {
		t.Fatalf("NewLeakyBucket couldnt initialse: %v", err)
	}

	const workers = 10
	const perWorker = 20
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perWorker {
				lb.Allow(1)
				time.Sleep(10 * time.Millisecond) // simulating some load
			}
		}()
	}

	wg.Wait()

	processed, dropped, _ := lb.Stats()
	totalHandled := processed + dropped
	expectedMin := int64(workers * perWorker / 10)

	if totalHandled < expectedMin {
		t.Errorf("Expected at least %d handled, got %d", expectedMin, totalHandled)
	}
}

// Basic testing
func ExampleLeakyBucket() {
	lb, _ := NewLeakyBucket(5, 1, PerSecond)

	for range 8 {
		if lb.Allow(1) {
			println("queued")
		} else {
			println("dropped")
		}
		time.Sleep(100 * time.Millisecond)
	}

}
