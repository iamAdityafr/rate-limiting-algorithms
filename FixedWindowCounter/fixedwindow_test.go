package main

import (
	"sync"
	"testing"
	"time"
)

// basic testing
func ExampleFixedWindowCounter() {
	fwc, _ := NewFixedWindowCounter(5*time.Second, 3)

	// Simulate 5 requests
	for range 5 {
		if fwc.Allow(1) {
			println("allowed")
		} else {
			println("denied")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestStats(t *testing.T) {
	fwc, err := NewFixedWindowCounter(5*time.Second, 3)
	if err != nil {
		t.Fatalf("NewFixedWindowCounter failed: %v", err)
	}

	fwc.Allow(2)
	fwc.Allow(2) // count: 2 , exceeds limit-> denied

	count, max, windowStart := fwc.Stats()
	if count != 2 {
		t.Errorf("Expected count=2, got %d", count)
	}
	if max != 3 {
		t.Errorf("Expected max=3, got %d", max)
	}

	now := time.Now()
	if windowStart.After(now) || windowStart.Before(now.Add(-10*time.Second)) {
		t.Errorf("Window start time seems wrong: %v", windowStart)
	}
}

func TestAllow(t *testing.T) {
	fwc, err := NewFixedWindowCounter(5*time.Second, 3)
	if err != nil {
		t.Fatalf("NewFixedWindowCounter failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		if !fwc.Allow(1) {
			t.Errorf("Allow(1) at %d should succeed", i)
		}
	}

	if fwc.Allow(1) {
		t.Error("Allow(1) after limit should fail")
	}

	time.Sleep(5*time.Second + 100*time.Millisecond)

	for i := 0; i < 2; i++ {
		if !fwc.Allow(1) {
			t.Errorf("Allow(1) after reset should succeed, iteration %d", i)
		}
	}
}

func TestConcurrency(t *testing.T) {
	fwc, err := NewFixedWindowCounter(10*time.Second, 50)
	if err != nil {
		t.Fatalf("NewFixedWindowCounter failed: %v", err)
	}

	const workers = 20
	const perWorker = 10
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				fwc.Allow(1)
				time.Sleep(10 * time.Millisecond) // simulate load
			}
		}()
	}

	wg.Wait()

	count, max, _ := fwc.Stats()

	if count > max {
		t.Errorf("Count %d should not exceed max %d", count, max)
	}
}

func TestWindowReset(t *testing.T) {
	fwc, err := NewFixedWindowCounter(1*time.Second, 2)
	if err != nil {
		t.Fatalf("NewFixedWindowCounter failed: %v", err)
	}

	fwc.Allow(2)

	count, max, _ := fwc.Stats()
	if count != max {
		t.Error("Should be at limit after using all requests")
	}

	time.Sleep(1100 * time.Millisecond)

	if !fwc.Allow(1) {
		t.Error("Should be able to allow after window reset")
	}
}

func TestTimeUntilReset(t *testing.T) {
	fwc, err := NewFixedWindowCounter(2*time.Second, 1)
	if err != nil {
		t.Fatalf("NewFixedWindowCounter failed: %v", err)
	}

	waitTime := fwc.TimeUntilReset()

	if waitTime < 0 || waitTime > 2*time.Second {
		t.Errorf("Wait time should be between 0 and 2s, got %v", waitTime)
	}

	time.Sleep(500 * time.Millisecond)
	newWaitTime := fwc.TimeUntilReset()

	if newWaitTime >= waitTime {
		t.Errorf("Wait time should decrease, was %v, now %v", waitTime, newWaitTime)
	}
}
