//go:build nats

package mesh

import (
	"context"
	"encoding/json"
	"time"

	nats "github.com/nats-io/nats.go"
)

type NatsBus struct {
	nc *nats.Conn
}

func NewNatsBus(url string) (*NatsBus, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NatsBus{nc: nc}, nil
}

func (b *NatsBus) Publish(ctx context.Context, e Event) error {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	payload, _ := json.Marshal(e)
	return b.nc.Publish(e.Topic, payload)
}

func (b *NatsBus) Subscribe(topic string, h Handler) (func(), error) {
	sub, err := b.nc.Subscribe(topic, func(msg *nats.Msg) {
		var e Event
		if err := json.Unmarshal(msg.Data, &e); err == nil {
			h(context.Background(), e)
		}
	})
	if err != nil {
		return nil, err
	}
	return func() { _ = sub.Unsubscribe() }, nil
}

func (b *NatsBus) Close() error { b.nc.Flush(); b.nc.Close(); return nil }
