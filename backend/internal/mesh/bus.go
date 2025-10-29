package mesh

import (
	"context"
	"encoding/json"
	"time"
)

const (
	TopicGraphInvalidate  = "graph.invalidate"
	TopicPolicyInvalidate = "policy.invalidate"
)

type Event struct {
	Topic     string          `json:"topic"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"ts"`
}

type Handler func(ctx context.Context, e Event)

type Bus interface {
	Publish(ctx context.Context, e Event) error
	Subscribe(topic string, h Handler) (unsubscribe func(), err error)
	Close() error
}
