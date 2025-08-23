package main

import (
	"errors"
	"log"
	"sync"
	"time"
)

type FixedWindowCounter struct {
	WindowSize      time.Duration
	MaxRequests     int64
	CurrentWindow   int64
	RequestCount    int64
	RequestsAllowed int64
	RequestsDenied  int64
	logger          *log.Logger
	mu              sync.RWMutex
}

func NewFixedWindowCounter(windowSize time.Duration, maxRequests int64) (*FixedWindowCounter, error) {
	if windowSize <= 0 {
		return nil, errors.New("window size must be positive")
	}
	if maxRequests <= 0 {
		return nil, errors.New("max requests must be positive")
	}

	windowStart := time.Now().Truncate(windowSize).Unix()

	return &FixedWindowCounter{
		WindowSize:      windowSize,
		MaxRequests:     maxRequests,
		CurrentWindow:   windowStart,
		RequestCount:    0,
		RequestsAllowed: 0,
		RequestsDenied:  0,
		logger:          log.Default(),
	}, nil
}

func (fwc *FixedWindowCounter) Allow(n int) bool {
	if n <= 0 {
		return false
	}

	fwc.mu.Lock()
	defer fwc.mu.Unlock()

	windowStart := time.Now().Truncate(fwc.WindowSize).Unix()

	// if new window then requestCount becomes 0
	if windowStart != fwc.CurrentWindow {
		fwc.CurrentWindow = windowStart
		fwc.RequestCount = 0
		if fwc.logger != nil {
			fwc.logger.Printf("window reset, new window starts at %s", time.Unix(windowStart, 0).Format("15:04:05"))
		}
	}

	// checking if allowing n requests would exceed the limit
	if fwc.RequestCount+int64(n) <= fwc.MaxRequests {
		fwc.RequestCount += int64(n)
		fwc.RequestsAllowed += int64(n)
		if fwc.logger != nil {
			fwc.logger.Printf("allowed %d requests, count: %d/%d", n, fwc.RequestCount, fwc.MaxRequests)
		}
		return true
	}

	fwc.RequestsDenied += int64(n)
	if fwc.logger != nil {
		fwc.logger.Printf("denied %d requests, limit exceeded: %d/%d", n, fwc.RequestCount, fwc.MaxRequests)
	}
	return false
}

func (fwc *FixedWindowCounter) SetLogger(logger *log.Logger) {
	fwc.mu.Lock()
	defer fwc.mu.Unlock()
	fwc.logger = logger
}

func (fwc *FixedWindowCounter) Stats() (currentCount, maxRequests int64, windowStart time.Time) {
	fwc.mu.RLock()
	defer fwc.mu.RUnlock()

	windowStartTime := time.Unix(fwc.CurrentWindow, 0)
	return fwc.RequestCount, fwc.MaxRequests, windowStartTime
}

func (fwc *FixedWindowCounter) Reset() {
	fwc.mu.Lock()
	defer fwc.mu.Unlock()

	fwc.CurrentWindow = time.Now().Truncate(fwc.WindowSize).Unix() // remove or comment this line if you choose to keep the current window boundary

	fwc.RequestCount = 0
	fwc.RequestsAllowed = 0
	fwc.RequestsDenied = 0

	if fwc.logger != nil {
		fwc.logger.Printf("stats reset")
	}
}

func (fwc *FixedWindowCounter) TimeUntilReset() time.Duration {
	fwc.mu.RLock()
	defer fwc.mu.RUnlock()

	windowStart := time.Unix(fwc.CurrentWindow, 0)
	windowEnd := windowStart.Add(fwc.WindowSize)
	return time.Until(windowEnd)
}
