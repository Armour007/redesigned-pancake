package api

import (
	"context"
	"encoding/json"

	"github.com/Armour007/aura-backend/internal/mesh"
)

var bus mesh.Bus

func SetBus(b mesh.Bus) { bus = b }
func getBus() mesh.Bus  { return bus }

func PublishGraphInvalidate(ctx context.Context) {
	if bus != nil {
		_ = bus.Publish(ctx, mesh.Event{Topic: mesh.TopicGraphInvalidate})
	}
}
func PublishPolicyInvalidate(ctx context.Context, policyID string) {
	if bus == nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{"policy_id": policyID})
	_ = bus.Publish(ctx, mesh.Event{Topic: mesh.TopicPolicyInvalidate, Payload: payload})
}
