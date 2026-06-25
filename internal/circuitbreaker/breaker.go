package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var ErrOpen = errors.New("circuit breaker open")

type Config struct {
	FailureThreshold    int
	OpenTimeout         time.Duration
	HalfOpenMaxRequests int
}

func (c Config) withDefaults() Config {
	if c.FailureThreshold <= 0 {
		c.FailureThreshold = 5
	}
	if c.OpenTimeout <= 0 {
		c.OpenTimeout = 30 * time.Second
	}
	if c.HalfOpenMaxRequests <= 0 {
		c.HalfOpenMaxRequests = 2
	}
	return c
}

type state int

const (
	stateClosed state = iota
	stateOpen
	stateHalfOpen
)

type Breaker struct {
	cfg Config

	mu              sync.Mutex
	state           state
	failures        int
	halfOpenSuccess int
	openedAt        time.Time
}

func New(cfg Config) *Breaker {
	return &Breaker{cfg: cfg.withDefaults(), state: stateClosed}
}

func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case stateClosed:
		return true
	case stateOpen:
		if time.Since(b.openedAt) >= b.cfg.OpenTimeout {
			b.state = stateHalfOpen
			b.halfOpenSuccess = 0
			return true
		}
		return false
	case stateHalfOpen:
		return b.halfOpenSuccess < b.cfg.HalfOpenMaxRequests
	default:
		return false
	}
}

func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case stateClosed:
		b.failures = 0
	case stateHalfOpen:
		b.halfOpenSuccess++
		if b.halfOpenSuccess >= b.cfg.HalfOpenMaxRequests {
			b.state = stateClosed
			b.failures = 0
			b.halfOpenSuccess = 0
		}
	}
}

func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case stateClosed:
		b.failures++
		if b.failures >= b.cfg.FailureThreshold {
			b.tripLocked()
		}
	case stateHalfOpen:
		b.tripLocked()
	}
}

func (b *Breaker) tripLocked() {
	b.state = stateOpen
	b.openedAt = time.Now()
	b.failures = 0
	b.halfOpenSuccess = 0
}

func (b *Breaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case stateClosed:
		return "closed"
	case stateOpen:
		return "open"
	case stateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}
