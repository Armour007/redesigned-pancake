package api

import (
	"time"

	"github.com/Armour007/aura-backend/internal/rel"
)

var graphClient rel.GraphClient

func SetGraphClient(gc rel.GraphClient) { graphClient = gc }
func getGraph() rel.GraphClient         { return graphClient }

// Helpers to build a cached client from an inner implementation
func NewCachedGraph(inner rel.GraphClient, ttl, negTtl time.Duration) rel.GraphClient {
	return rel.NewCachedGraph(inner, ttl, negTtl)
}

// ClearGraphCache clears the underlying cache if the client is a CachedGraph wrapper
func ClearGraphCache() {
	if cg, ok := graphClient.(*rel.CachedGraph); ok {
		cg.Clear()
	}
}
