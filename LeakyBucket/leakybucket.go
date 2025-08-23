package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

type LeakyBucket struct {
	capacity          int64
	leakRate          float64
	queue             float64
	requestsDropped   int64
	requestsProcessed int64
	logger            *log.Logger
	lastLeakTime      time.Time
	mutex             sync.RWMutex
}

type TimeUnit string

const (
	PerSecond TimeUnit = "second"
	PerMinute          = "minute"
	PerHour            = "hour"
)

func NewLeakyBucket(capacity int64, leakRate float64, unit TimeUnit) (*LeakyBucket, error) {
	if capacity <= 0 || leakRate <= 0 {
		return nil, errors.New("capacity and leakRate must be positive")
	}

	ratePerSecond := leakRate
	switch unit {
	case PerSecond:

	case PerMinute:
		ratePerSecond = leakRate / 60.0
	case PerHour:
		ratePerSecond = leakRate / 3600.0

	default:
		return nil, errors.New("invalid time unit")
	}

	return &LeakyBucket{
		capacity:     capacity,
		leakRate:     ratePerSecond,
		queue:        0,
		lastLeakTime: time.Now(),
		logger:       log.Default(),
	}, nil
}

func (lb *LeakyBucket) getCurrentQueue() float64 {
	elapsed := time.Since(lb.lastLeakTime).Seconds()

	leaked := elapsed * lb.leakRate
	currentQueue := lb.queue - leaked

	if currentQueue < 0 {
		currentQueue = 0
	}

	return currentQueue
}

func (lb *LeakyBucket) leak() {
	elapsed := time.Since(lb.lastLeakTime).Seconds()

	if elapsed <= 0 {
		return
	}

	leaked := elapsed * lb.leakRate
	if leaked >= lb.queue {
		lb.requestsProcessed += int64(lb.queue)
		lb.queue = 0
	} else {
		lb.requestsProcessed += int64(leaked)
		lb.queue -= leaked
	}
	lb.lastLeakTime = time.Now()
}

func (lb *LeakyBucket) Allow(n int) bool {
	if n <= 0 {
		return false
	}

	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.leak()

	if lb.queue+float64(n) > float64(lb.capacity) {
		lb.requestsDropped += int64(n)
		if lb.logger != nil {
			lb.logger.Printf("dropped %d requests, queue full -> %.2f/%d", n, lb.queue, lb.capacity)
		}
		return false
	}

	lb.queue += float64(n)
	if lb.logger != nil {
		lb.logger.Printf("queued %d requests, queue size -> %.2f/%d", n, lb.queue, lb.capacity)
	}
	return true

}

func (lb *LeakyBucket) Take(ctx context.Context, n int) error {
	if n <= 0 {
		return errors.New("n must be positive")
	}
	for {
		if lb.Allow(n) {
			return nil
		}
		lb.mutex.RLock()
		waitTime := lb.TimeUntilSpace(n)
		lb.mutex.RUnlock()

		if waitTime <= 0 {
			continue
		}

		timer := time.NewTimer(waitTime)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C // cleaning up the signal
			}
			return ctx.Err()
		case <-timer.C:
		}
	}

}

func (lb *LeakyBucket) TimeUntilSpace(n int) time.Duration {
	currentQueue := lb.getCurrentQueue()

	if currentQueue+float64(n) <= float64(lb.capacity) {
		return 0
	}

	spaceNeeded := currentQueue + float64(n) - float64(lb.capacity)
	secondsNeeded := spaceNeeded / lb.leakRate

	return time.Duration(secondsNeeded * float64(time.Second))

}
func (lb *LeakyBucket) TimeUntilAllowed(n int) time.Duration {
	if n <= 0 {
		return 0
	}

	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	return lb.TimeUntilSpace(n)
}

func (lb *LeakyBucket) QueueSize() float64 {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	return lb.getCurrentQueue()
}

func (lb *LeakyBucket) Capacity() int64 {
	return lb.capacity
}

func (lb *LeakyBucket) LeakRate() float64 {
	return lb.leakRate
}

func (lb *LeakyBucket) Stats() (processed, dropped int64, queueSize float64) {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	return lb.requestsProcessed, lb.requestsDropped, lb.getCurrentQueue()
}

func (lb *LeakyBucket) ResetStats() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	lb.requestsProcessed = 0
	lb.requestsDropped = 0
}

func (lb *LeakyBucket) SetLogger(logger *log.Logger) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	lb.logger = logger
}

func (lb *LeakyBucket) IsEmpty() bool {
	return lb.QueueSize() == 0
}

func (lb *LeakyBucket) IsFull() bool {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()
	return lb.getCurrentQueue() >= float64(lb.capacity)
}
