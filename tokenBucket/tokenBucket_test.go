package main

import (
	"testing"
	"time"
)

func TestAllow(t *testing.T) {
	tb, err := NewTokenBucket(10, 10, 1)
	if err != nil {
		t.Fatalf("token bucket couldnt initialised: %v", err)
	}

	for i := 0; i < 10; i++ {
		if !tb.Allow(1) {
			t.Errorf("Allow(1) at %d should succeed", i)
		}
	}

	if tb.Allow(1) {

		t.Errorf("Allow(1) after limit should fail")
	}

	time.Sleep(5 * time.Second)
	for i := 0; i < 5; i++ {
		if !tb.Allow(1) {
			t.Errorf("Allow(1) after %d leak should succeed", i)
		}
	}

	if tb.Allow(1) {
		t.Errorf("Allow(1) after using refilled tokens should fail")
	}

	// allowing more than refill should fail
	tb, _ = NewTokenBucket(10, 0, 1)
	time.Sleep(5 * time.Second)
	if tb.Allow(6) {
		t.Errorf("Allow(6) with only 5 tokens available should fail")
	}

}

func TestTimeUntilSpace(t *testing.T) {
	tb, _ := NewTokenBucket(10, 5, 2)

	if delay := tb.TimeUntilAllowed(3); delay != 0 {
		t.Errorf("Expected delay 0 for 3 tokens, but got: %v", delay)
	}

}

func TestWaitAllow(t *testing.T) {
	tb, _ := NewTokenBucket(10, 0, 2)

	start := time.Now()
	ok := tb.WaitAllow(2, 3*time.Second)
	elapsed := time.Since(start)

	if !ok {
		t.Errorf("it shouldve succeed within")
	}
	if elapsed < 2*time.Second {
		t.Errorf("happened too early: %v", elapsed)
	}
	if elapsed > 3*time.Second {
		t.Errorf("took too long to happen: %v", elapsed)
	}

	// Requesting 0 tokens and should fail
	tb1, _ := NewTokenBucket(10, 10, 1)
	if tb1.WaitAllow(0, 1*time.Second) {
		t.Errorf("WaitAllow(0) should return false")
	}

	// Requesting more tokens than capacity
	tb2, _ := NewTokenBucket(10, 10, 1)
	if tb2.WaitAllow(11, 1*time.Second) {
		t.Errorf("WaitAllow(11) should fail")
	}
}

func TestUpdate(t *testing.T) {
	tb, _ := NewTokenBucket(10, 10, 1)
	//updating capacity only
	tb.Update(5, -1)
	if tb.capacity != 5 {
		t.Errorf("Expected tokens to set 5 but got: %v", tb.capacity)
	}
	if tb.fillRate != 1 {
		t.Errorf("Expected tokens to stay 1 but got: %v", tb.fillRate)
	}

	// updating fillRate only
	old := tb.capacity
	tb.Update(0, 2)
	if tb.fillRate != 2 {
		t.Errorf("Expected fillRate 2, got %f", tb.fillRate)
	}
	if tb.capacity != old {
		t.Errorf("Capacity should remain %d, got %d", old, tb.capacity)
	}

	// updating both capacity and fillRate
	tb.Update(20, 1)
	if tb.capacity != 20 || tb.fillRate != 1 {
		t.Errorf("Expected capacity 20 and fillRate 1, got %d and %f", tb.capacity, tb.fillRate)
	}

}

func TestStats(t *testing.T) {
	tb, _ := NewTokenBucket(10, 10, 1)
	tb.Allow(11)

	processed, rejected := tb.Stats()
	if processed != 0 {
		t.Errorf("Expected 1 to be processed, but got %v", processed)
	}
	if rejected != 1 {
		t.Errorf("Expected 1 to be rejected, but got %v", rejected)
	}

	tb.ResetStats()
	processed, rejected = tb.Stats()
	if processed != 0 || rejected != 0 {
		t.Errorf("After reset: got %d,%d", processed, rejected)
	}
}
