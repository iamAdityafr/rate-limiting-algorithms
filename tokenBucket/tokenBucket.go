package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

type TokenBucket struct {
	capacity        int
	fillRate        float64
	tokens          float64
	lastTime        time.Time
	tokensProcessed int
	tokensRejected  int
	log             *log.Logger
	mu              sync.RWMutex
}

func NewTokenBucket(capacity int, tokens, fillRate float64) (*TokenBucket, error) {
	if tokens < 0 {
		return nil, errors.New("tokens cant be negative")
	}
	if fillRate <= 0 {
		return nil, errors.New("fillRate cant be negative")
	}
	if tokens > float64(capacity) {
		tokens = float64(capacity)
	}
	now := time.Now()
	return &TokenBucket{
		capacity:        capacity,
		fillRate:        fillRate,
		tokens:          tokens,
		tokensProcessed: 0,
		tokensRejected:  0,
		lastTime:        now,
		log:             log.Default(),
	}, nil
}
func (tb *TokenBucket) SetLogger(logger *log.Logger) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if logger != nil {
		tb.log = logger
	}
}
func (tb *TokenBucket) Allow(n int) bool {
	if n <= 0 {
		return false
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if float64(n) > tb.tokens {
		tb.tokensRejected++
		tb.log.Printf("Rejected: need %d tokens, have %.2f", n, tb.tokens)
		return false
	}

	tb.tokens -= float64(n)
	tb.tokensProcessed++
	return true
}

func (tb *TokenBucket) Update(newCapacity int, newFillRate float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if newCapacity > 0 {
		tb.capacity = newCapacity
		if tb.tokens > float64(newCapacity) {
			tb.tokens = float64(newCapacity)
		}
	}
	if newFillRate > 0 {
		tb.fillRate = newFillRate
	}
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	timeElapsed := now.Sub(tb.lastTime).Seconds()

	const maxRefillSeconds = 60
	if timeElapsed > maxRefillSeconds {
		timeElapsed = maxRefillSeconds
	}

	tb.tokens += timeElapsed * tb.fillRate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
	tb.lastTime = now
}

func (tb *TokenBucket) TimeUntilAllowed(n int) time.Duration {
	if n <= 0 {
		return 0
	}
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	return tb.TimeUntilSpace(n)
}

func (tb *TokenBucket) TimeUntilSpace(n int) time.Duration {

	now := time.Now()
	timeElapsed := now.Sub(tb.lastTime).Seconds()

	const maxRefillSeconds = 60
	if timeElapsed > maxRefillSeconds {
		timeElapsed = maxRefillSeconds
	}

	currentTokens := tb.tokens + (timeElapsed * tb.fillRate)
	if currentTokens > float64(tb.capacity) {
		currentTokens = float64(tb.capacity)
	}

	missing := float64(n) - currentTokens
	if missing <= 0 {
		return 0
	}

	seconds := missing / tb.fillRate
	return time.Duration(seconds * float64(time.Second))
}

func (tb *TokenBucket) WaitAllow(n int, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return tb.WaitAllowContext(ctx, n)
}

func (tb *TokenBucket) WaitAllowContext(ctx context.Context, n int) bool {
	if n <= 0 {
		return false
	}

	const maxSleep = 100 * time.Millisecond
	ticker := time.NewTicker(maxSleep)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			tb.mu.Lock()
			tb.refill()

			if float64(n) <= tb.tokens {
				tb.tokens -= float64(n)
				tb.tokensProcessed++
				tb.mu.Unlock()
				return true
			}
			tb.mu.Unlock()
		}
	}
}

func (tb *TokenBucket) Stats() (processed, rejected int) {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.tokensProcessed, tb.tokensRejected
}

func (tb *TokenBucket) ResetStats() {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.tokensProcessed = 0
	tb.tokensRejected = 0
}
func (tb *TokenBucket) AvailableTokens() float64 {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.tokens
}
