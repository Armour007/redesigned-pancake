package mesh

import (
	"context"
	"sync"
	"time"
)

type LocalBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewLocalBus() *LocalBus { return &LocalBus{handlers: map[string][]Handler{}} }

func (b *LocalBus) Publish(ctx context.Context, e Event) error {
	b.mu.RLock()
	hs := append([]Handler(nil), b.handlers[e.Topic]...)
	b.mu.RUnlock()
	e.Timestamp = time.Now()
	// fan out asynchronously in goroutines
	for _, h := range hs {
		go h(ctx, e)
	}
	return nil
}

func (b *LocalBus) Subscribe(topic string, h Handler) (func(), error) {
	b.mu.Lock()
	b.handlers[topic] = append(b.handlers[topic], h)
	idx := len(b.handlers[topic]) - 1
	b.mu.Unlock()
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		hs := b.handlers[topic]
		if idx >= 0 && idx < len(hs) {
			b.handlers[topic] = append(hs[:idx], hs[idx+1:]...)
		}
	}, nil
}

func (b *LocalBus) Close() error { return nil }
