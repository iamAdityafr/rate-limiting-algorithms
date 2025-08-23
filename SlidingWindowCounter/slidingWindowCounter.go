package main

import (
	"errors"
	"log"
	"sync"
	"time"
)

type SlidingWindow struct {
	windowSize            time.Duration
	maxRequests           int64
	currentWindow         int64
	lastWindowRequests    int64
	currentWindowRequests int64
	RequestCount          int64
	PrevCount             int64
	Log                   *log.Logger
	mu                    sync.RWMutex
	requestsAllowed       int64
	requestsDenied        int64
	logger                *log.Logger
}

func NewSlidingWindow(windowSize time.Duration, maxRequests int64) (*SlidingWindow, error) {
	if windowSize <= 0 {
		return nil, errors.New("window size can't be negative")
	}
	if maxRequests <= 0 {
		return nil, errors.New("max requests can't be negative")
	}

	now := time.Now()
	truncated := now.Truncate(windowSize)
	currentWindowNano := truncated.UnixNano()

	return &SlidingWindow{
		windowSize:            windowSize,
		maxRequests:           maxRequests,
		currentWindow:         currentWindowNano,
		lastWindowRequests:    0,
		currentWindowRequests: 0,
		requestsAllowed:       0,
		requestsDenied:        0,
		logger:                log.Default(),
	}, nil
}

func (sw *SlidingWindow) Allow(n int) bool {
	if n <= 0 {
		return false
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	currentWindowStart := now.Truncate(sw.windowSize)
	currentWindowNano := currentWindowStart.UnixNano()

	if currentWindowNano != sw.currentWindow {
		sw.lastWindowRequests = sw.currentWindowRequests
		sw.currentWindowRequests = 0
		sw.currentWindow = currentWindowNano

		if sw.logger != nil {
			sw.logger.Printf("window shifeted: last window requests= %d, new window = %s", sw.lastWindowRequests, time.Unix(currentWindowNano, 0).Format("15:04:05")) // for "23:59:59" style
		}
	}
	windowStart := time.Unix(0, currentWindowNano).UTC()
	windowElapsed := now.Sub(windowStart)
	if windowElapsed < 0 {
		windowElapsed = 0
	}
	doneRatio := float64(windowElapsed) / float64(sw.windowSize)
	if doneRatio > 1.0 {
		doneRatio = 1.0
	}

	carryOver := float64(sw.lastWindowRequests)
	sliding := float64(sw.currentWindowRequests) + (1.0-doneRatio)*carryOver // sliding window formula

	if sliding+float64(n) <= float64(sw.maxRequests) {
		sw.currentWindowRequests += int64(n)
		sw.requestsAllowed += int64(n)

		if sw.logger != nil {
			sw.logger.Printf("allowed %d requests: sliding count = %.2f, current = %d, last = %d, done = %.2f", n, sliding, sw.currentWindowRequests, sw.lastWindowRequests, doneRatio)
		}
		return true
	}

	sw.requestsDenied += int64(n)
	if sw.logger != nil {
		sw.logger.Printf("denied %d requests: sliding count = %.2f (exceed limit = %d)", n, sliding+float64(n), sw.maxRequests)
	}
	return false
}

func (sw *SlidingWindow) Stats() (currentCount float64, maxRequests int64, windowStart time.Time) {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	windowStart = time.Unix(0, sw.currentWindow).UTC()

	now := time.Now()
	windowElapsed := now.Sub(windowStart)
	doneRatio := float64(windowElapsed) / float64(sw.windowSize)

	if doneRatio > 1.0 {
		doneRatio = 1.0
	}

	sliding := float64(sw.currentWindowRequests) + (1.0-doneRatio)*float64(sw.lastWindowRequests)

	return sliding, sw.maxRequests, windowStart
}

func (sw *SlidingWindow) Reset() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	truncated := now.Truncate(sw.windowSize)
	sw.currentWindow = truncated.UnixNano()

	sw.requestsDenied = 0
	sw.requestsAllowed = 0
	sw.currentWindowRequests = 0
	sw.lastWindowRequests = 0

	if sw.logger != nil {
		sw.logger.Printf("stats reset")
	}
}

func (sw *SlidingWindow) SetLogger(logger *log.Logger) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.logger = logger
}

func (sw *SlidingWindow) TimeUntilAllowed(n int) time.Duration {
	if n <= 0 {
		return 0
	}

	sw.mu.RLock()
	defer sw.mu.RUnlock()

	windowStart := time.Unix(0, sw.currentWindow).UTC()
	now := time.Now()
	elapsed := now.Sub(windowStart)

	if elapsed >= sw.windowSize {
		return 0
	}

	doneRatio := float64(elapsed) / float64(sw.windowSize)
	slidingCount := float64(sw.currentWindowRequests) + (1.0-doneRatio)*float64(sw.lastWindowRequests)

	if slidingCount+float64(n) <= float64(sw.maxRequests) {
		return 0
	}

	// Waiting until current window ends
	return sw.windowSize - elapsed
}

// stats for metrics
func (sw *SlidingWindow) DetailedStats() (allowed, denied int64, currentSlidingCount float64) {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	now := time.Now()
	currentWindow := time.Unix(0, sw.currentWindow).UTC()
	windowElapsed := now.Sub(currentWindow)
	doneRatio := float64(windowElapsed) / float64(sw.windowSize)

	if doneRatio > 1.0 {
		doneRatio = 1.0
	}

	slidingCount := float64(sw.currentWindowRequests) + (1.0-doneRatio)*float64(sw.lastWindowRequests)

	return sw.requestsAllowed, sw.requestsDenied, slidingCount
}
