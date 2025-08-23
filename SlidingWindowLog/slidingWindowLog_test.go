package main

import (
	"testing"
	"time"
)

func TestSlidingBasic(t *testing.T) {
	swl, _ := NewSlidingWindowLog(100*time.Millisecond, 3)

	if !swl.Allow(3) {
		t.Errorf("Expected 3 request to be allowed")
	}

	if swl.Allow(1) {
		t.Errorf("expected 4th request to be denied")
	}

	time.Sleep(100 * time.Millisecond)
	if !swl.Allow(1) {
		t.Errorf("expected request to be allowed after window expired")
	}
}

func TestStats(t *testing.T) {
	swl, _ := NewSlidingWindowLog(time.Minute, 5)

	count, max, _ := swl.Stats()
	if count != 0 || max != 5 {
		t.Errorf("Expected count=0, max=5 but got count=%d, max=%d", count, max)
	}

	swl.Allow(3)
	count, max, _ = swl.Stats()
	if count != 3 || max != 5 {
		t.Errorf("Expected count=3, max=5, got count=%d, max=%d", count, max)
	}
}

func TestReset(t *testing.T) {
	swl, _ := NewSlidingWindowLog(time.Minute, 5)
	swl.Allow(5)
	if swl.Allow(1) {
		t.Errorf("should be denied before reseting")
	}

	swl.Reset()
	if !swl.Allow(5) {
		t.Errorf("after reseting should be allowed")
	}
}

func TestAllow(t *testing.T) {
	swl, _ := NewSlidingWindowLog(100*time.Millisecond, 5)

	for i := range 5 {
		if !swl.Allow(1) {
			t.Errorf("Allow(1) at %d should succeed", i)
		}
	}
	if swl.Allow(1) {
		t.Error("Allow(1) should fail after full")
	}

	time.Sleep(140 * time.Millisecond)
	for i := range 3 {
		if !swl.Allow(1) {
			t.Errorf("Allow(1) after %d leak should succeed", i)
		}
	}

}

func TestTimeUntilAllowed(t *testing.T) {
	swl, _ := NewSlidingWindowLog(100*time.Millisecond, 5)

	if delay := swl.TimeUntilAllowed(1); delay != 0 {
		t.Errorf("Expected no delay, got %v", delay)
	}

	swl.Allow(5)
	if delay := swl.TimeUntilAllowed(1); delay <= 0 {
		t.Errorf("should be positive delay but got: %v", delay)
	}
}
