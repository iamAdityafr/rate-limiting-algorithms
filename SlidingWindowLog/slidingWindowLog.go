package main

import (
	"errors"
	"log"
	"sync"
	"time"
)

type SlidingWindowLog struct {
	windowSize  time.Duration
	maxRequests int64
	requestLog  *Deque[time.Time] // storing timestamps of requests
	logger      *log.Logger
	mu          sync.RWMutex
}

func NewSlidingWindowLog(windowSize time.Duration, maxRequests int64) (*SlidingWindowLog, error) {
	if windowSize <= 0 {
		return nil, errors.New("window size must be positive")
	}
	if maxRequests <= 0 {
		return nil, errors.New("max requests must be positive")
	}

	return &SlidingWindowLog{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		requestLog:  NewDeque[time.Time](),
		logger:      log.Default(),
	}, nil
}

func (sw *SlidingWindowLog) Allow(n int) bool {
	if n <= 0 {
		return false
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.windowSize) // doing minus here to go back in time by window size

	// removing expired requests from front
	for {
		if sw.requestLog.IsEmpty() {
			break
		}

		frontTime, exists := sw.requestLog.PeekFront()
		if !exists {
			break
		}
		if frontTime.After(windowStart) {
			break
		}

		// Removing expired request
		sw.requestLog.PopFront()
	}

	if int64(sw.requestLog.Size())+int64(n) <= sw.maxRequests {
		for range n {
			sw.requestLog.PushBack(now)
		}

		if sw.logger != nil {
			sw.logger.Printf("allowed %d requests, current count: %d/%d",
				n, sw.requestLog.Size(), sw.maxRequests)
		}
		return true
	}

	if sw.logger != nil {
		sw.logger.Printf("denied %d requests, limit exceeded: %d/%d",
			n, sw.requestLog.Size(), sw.maxRequests)
	}
	return false
}

func (sw *SlidingWindowLog) Stats() (currentCount, maxRequests int64, windowStart time.Time) {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	now := time.Now()
	windowStart = now.Add(-sw.windowSize)

	return int64(sw.requestLog.Size()), sw.maxRequests, windowStart
}

func (sw *SlidingWindowLog) TimeUntilAllowed(n int) time.Duration {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	if int64(sw.requestLog.Size())+int64(n) <= sw.maxRequests {
		return 0
	}

	// Find the oldest request that would need to expire
	requestsToRemove := int64(sw.requestLog.Size()) + int64(n) - sw.maxRequests

	if requestsToRemove >= int64(sw.requestLog.Size()) {
		// Need to wait for all current requests to expire
		if sw.requestLog.IsEmpty() {
			return 0
		}
		oldestRequest, exists := sw.requestLog.PeekFront()
		if !exists {
			return 0
		}
		return time.Until(oldestRequest.Add(sw.windowSize))
	}

	return sw.windowSize
}

func (sw *SlidingWindowLog) Reset() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// clearing the deque
	sw.requestLog = NewDeque[time.Time]()

	if sw.logger != nil {
		sw.logger.Printf("sliding window reset")
	}
}

func (sw *SlidingWindowLog) SetLogger(logger *log.Logger) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.logger = logger
}

func (sw *SlidingWindowLog) GetWindowSize() time.Duration {
	return sw.windowSize
}

func (sw *SlidingWindowLog) GetMaxRequests() int64 {
	return sw.maxRequests
}
