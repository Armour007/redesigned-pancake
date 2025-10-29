package risk

import (
	"sync"
	"time"
)

// Signals represents computed runtime signals for a subject (org,agent)
// Presently includes a 0-100 risk score and anomaly flags.
// Simple prototype based on request rate in a sliding window.

type Signals struct {
	Score int      `json:"score"`
	Flags []string `json:"flags"`
}

type window struct {
	mu  sync.Mutex
	buf []time.Time
}

type Tracker struct {
	mu        sync.Mutex
	byKey     map[string]*window
	winDur    time.Duration
	threshold int
	alerts    map[string]map[string]time.Time // key -> flag -> expiresAt
}

// NewTracker returns a sliding-window tracker.
// window: duration to look back; threshold: requests within window that map to score 100.
func NewTracker(win time.Duration, threshold int) *Tracker {
	if threshold <= 0 {
		threshold = 50
	}
	if win <= 0 {
		win = 30 * time.Second
	}
	return &Tracker{byKey: make(map[string]*window), winDur: win, threshold: threshold, alerts: make(map[string]map[string]time.Time)}
}

func (t *Tracker) key(orgID, agentID string) string { return orgID + ":" + agentID }

// Record increments the hit counter for the given org/agent at now.
func (t *Tracker) Record(orgID, agentID string, now time.Time) {
	k := t.key(orgID, agentID)
	w := t.getWindow(k)
	w.mu.Lock()
	w.buf = append(w.buf, now)
	w.mu.Unlock()
}

// Get returns current signals for org/agent.
func (t *Tracker) Get(orgID, agentID string, now time.Time) Signals {
	k := t.key(orgID, agentID)
	w := t.getWindow(k)
	// prune
	cut := now.Add(-t.winDur)
	w.mu.Lock()
	buf := w.buf
	i := 0
	for _, ts := range buf {
		if ts.After(cut) {
			buf[i] = ts
			i++
		}
	}
	buf = buf[:i]
	w.buf = buf
	count := len(buf)
	w.mu.Unlock()

	score := 0
	flags := []string{}
	if count > 0 {
		score = count * 100 / t.threshold
		if score > 100 {
			score = 100
		}
		if count >= t.threshold {
			flags = append(flags, "rate_spike")
		}
	}
	// include active alerts
	t.mu.Lock()
	if m, ok := t.alerts[k]; ok {
		for name, exp := range m {
			if now.Before(exp) {
				flags = append(flags, name)
			}
		}
	}
	t.mu.Unlock()
	return Signals{Score: score, Flags: flags}
}

func (t *Tracker) getWindow(k string) *window {
	t.mu.Lock()
	defer t.mu.Unlock()
	w := t.byKey[k]
	if w == nil {
		w = &window{}
		t.byKey[k] = w
	}
	return w
}

// RaiseAlert sets a transient alert flag for org/agent until expiresAt
func (t *Tracker) RaiseAlert(orgID, agentID, flag string, expiresAt time.Time) {
	k := t.key(orgID, agentID)
	t.mu.Lock()
	defer t.mu.Unlock()
	m := t.alerts[k]
	if m == nil {
		m = make(map[string]time.Time)
		t.alerts[k] = m
	}
	m[flag] = expiresAt
}

// ClearAlert removes a flag for org/agent
func (t *Tracker) ClearAlert(orgID, agentID, flag string) {
	k := t.key(orgID, agentID)
	t.mu.Lock()
	defer t.mu.Unlock()
	if m, ok := t.alerts[k]; ok {
		delete(m, flag)
	}
}
