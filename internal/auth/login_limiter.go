package auth

import (
	"sync"
	"time"
)

const (
	loginAttemptLimit    = 5
	loginAttemptWindow   = time.Minute
	loginLimiterCapacity = 4096
)

type loginAttemptKey struct {
	clientIP string
	email    string
}

type loginAttemptWindowState struct {
	started time.Time
	count   int
}

// loginLimiter is a bounded, process-local fixed-window limiter. It limits a
// client/email pair without creating durable account lockouts.
type loginLimiter struct {
	mu       sync.Mutex
	attempts map[loginAttemptKey]loginAttemptWindowState
	limit    int
	window   time.Duration
	capacity int
	now      func() time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{
		attempts: make(map[loginAttemptKey]loginAttemptWindowState),
		limit:    loginAttemptLimit,
		window:   loginAttemptWindow,
		capacity: loginLimiterCapacity,
		now:      time.Now,
	}
}

func (limiter *loginLimiter) allow(key loginAttemptKey) bool {
	now := limiter.now()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	state, exists := limiter.attempts[key]
	if !exists {
		if len(limiter.attempts) >= limiter.capacity {
			limiter.removeOldest()
		}
		limiter.attempts[key] = loginAttemptWindowState{started: now, count: 1}
		return true
	}
	if now.Sub(state.started) >= limiter.window {
		limiter.attempts[key] = loginAttemptWindowState{started: now, count: 1}
		return true
	}
	if state.count >= limiter.limit {
		return false
	}
	state.count++
	limiter.attempts[key] = state
	return true
}

func (limiter *loginLimiter) reset(key loginAttemptKey) {
	limiter.mu.Lock()
	delete(limiter.attempts, key)
	limiter.mu.Unlock()
}

func (limiter *loginLimiter) removeOldest() {
	var oldestKey loginAttemptKey
	var oldest time.Time
	for key, state := range limiter.attempts {
		if oldest.IsZero() || state.started.Before(oldest) {
			oldestKey = key
			oldest = state.started
		}
	}
	delete(limiter.attempts, oldestKey)
}
