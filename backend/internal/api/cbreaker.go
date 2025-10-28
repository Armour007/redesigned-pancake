package api

import (
	"os"
	"strconv"
	"sync"
	"time"
)

type CircuitBreaker struct {
	name       string
	mu         sync.Mutex
	failures   int
	openedTill time.Time
	threshold  int
	openFor    time.Duration
	open       bool
}

var (
	breakersMu         sync.Mutex
	breakers           = map[string]*CircuitBreaker{}
	cbDefaultThreshold = envInt("AURA_CB_THRESHOLD", 3)
	cbDefaultOpenSec   = envInt("AURA_CB_OPEN_SECONDS", 30)
)

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func GetBreaker(name string) *CircuitBreaker {
	breakersMu.Lock()
	defer breakersMu.Unlock()
	if b, ok := breakers[name]; ok {
		return b
	}
	b := &CircuitBreaker{name: name, threshold: cbDefaultThreshold, openFor: time.Duration(cbDefaultOpenSec) * time.Second}
	breakers[name] = b
	// expose initial state
	SetBreakerState(name, false)
	return b
}

func (b *CircuitBreaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if time.Now().Before(b.openedTill) {
		b.open = true
		SetBreakerState(b.name, true)
		return false
	}
	if b.open { // transition to closed
		b.open = false
		SetBreakerState(b.name, false)
	}
	return true
}

func (b *CircuitBreaker) ReportSuccess() {
	b.mu.Lock()
	b.failures = 0
	if b.open {
		b.open = false
		SetBreakerState(b.name, false)
	}
	b.mu.Unlock()
}

func (b *CircuitBreaker) ReportFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.threshold {
		b.openedTill = time.Now().Add(b.openFor)
		b.failures = 0
		b.open = true
		SetBreakerState(b.name, true)
	}
}
