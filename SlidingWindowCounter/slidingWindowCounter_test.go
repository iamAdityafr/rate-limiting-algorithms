package main

import (
	"testing"
	"time"
)

func TestAllow(t *testing.T) {
	swc, err := NewSlidingWindow(100*time.Millisecond, 5)
	if err != nil {
		t.Fatalf("Failed to create sliding window: %v", err)
	}

	if !swc.Allow(3) {
		t.Error("Expected Allow(3) to succeed")
	}
	if !swc.Allow(2) {
		t.Error("Expected Allow(2) to succeed (total = 5)")
	}

	if swc.Allow(1) {
		t.Error("Expected Allow(1) to fail — limit exceeded")
	}

	time.Sleep(150 * time.Millisecond)

	if !swc.Allow(4) {
		t.Error("Expected Allow(4) to succeed in new window")
	}
}

func TestStats(t *testing.T) {
	swc, _ := NewSlidingWindow(time.Minute, 10)

	count, max, _ := swc.Stats()
	if count != 0 || max != 10 {
		t.Errorf("expecting count = 0, maxRequests = 10 but got count=%.2f, maxRequests=%d", count, max)
	}

	swc.Allow(5)
	count, max, _ = swc.Stats()
	if count != 5 || max != 10 {
		t.Errorf("Expected count=5, max=10, got count=%.2f, max=%d", count, max)
	}
}

func TestDetailedStats(t *testing.T) {
	swc, _ := NewSlidingWindow(time.Minute, 10)

	allowed, denied, sliding := swc.DetailedStats()
	if allowed != 0 || denied != 0 || sliding != 0 {
		t.Errorf("expected initial stats to be 0 but got allowed=%d , denied=%d, slidingcount=%.2f", allowed, denied, sliding)
	}

	swc.Allow(5)
	allowed, denied, sliding = swc.DetailedStats()
	if allowed != 5 || denied != 0 || sliding != 5 {
		t.Errorf("Expected allowed=5, denied=0, sliding=5 but got allowed=%d, denied=%d, sliding count=%.2f", allowed, denied, sliding)
	}
	swc.Allow(5)
	swc.Allow(1)
	allowed, denied, _ = swc.DetailedStats()
	if allowed != 10 || denied != 1 {
		t.Errorf("Expected allowed=10, denied but got allowed=%d, denied=%d", allowed, denied)
	}
}

func TestReset(t *testing.T) {
	swc, _ := NewSlidingWindow(time.Minute, 5)

	swc.Allow(5)
	swc.Allow(1)
	allowed, denied, _ := swc.DetailedStats()
	if allowed != 5 || denied != 1 {
		t.Errorf("Before reset: expected allowed = 5, denied = 1 but got allowed = %d, denied = %d", allowed, denied)
	}

	swc.Reset()
	allowed, denied, sliding := swc.DetailedStats()
	if allowed != 0 || denied != 0 || sliding != 0 {
		t.Errorf("After reset: expected all stats = 0 but got allowed=%d, denied=%d, sliding count=%.2f", allowed, denied, sliding)
	}

	if !swc.Allow(2) {
		t.Errorf("Should allow requests after reset")
	}
}
func TestWindow(t *testing.T) {
	swc, _ := NewSlidingWindow(100*time.Millisecond, 5)
	swc.Allow(5)

	if swc.Allow(1) {
		t.Errorf("Should be denied when window is full")
	}

	time.Sleep(150 * time.Millisecond)

	if !swc.Allow(4) {
		t.Errorf("Should allow requests in new window")
	}
}
func TestInvalidRequests(t *testing.T) {
	swc, _ := NewSlidingWindow(time.Minute, 5)

	if swc.Allow(0) {
		t.Errorf("Allow(0) should return false")
	}

	if swc.Allow(-1) {
		t.Errorf("Allow(-1) should return false")
	}
}
func TestUntilAllowed(t *testing.T) {
	swc, _ := NewSlidingWindow(100*time.Millisecond, 5)

	if delay := swc.TimeUntilAllowed(1); delay != 0 {
		t.Errorf("Expected no delay when under limit, got %v", delay)
	}

	swc.Allow(5)

	delay := swc.TimeUntilAllowed(1)
	if delay < 0 {
		t.Errorf("Expected positive delay but got %v", delay)
	}
	if delay > swc.windowSize {
		t.Errorf("Expected delay be less than window size=%v but got %v", swc.windowSize, delay)
	}

	if delay := swc.TimeUntilAllowed(0); delay != 0 {
		t.Errorf("Expected 0 delay for invalid input, got %v", delay)
	}
}
