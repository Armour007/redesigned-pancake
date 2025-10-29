package rel

import (
	"context"
	"time"
)

// RelationRef identifies a node in the graph: namespace:objectID (e.g., agent:123)
type RelationRef struct {
	Namespace string `json:"namespace"`
	ObjectID  string `json:"object_id"`
}

// GraphExpansion is a simplified expansion tree for debugging
type GraphExpansion struct {
	Relation string           `json:"relation"`
	Object   RelationRef      `json:"object"`
	Children []GraphExpansion `json:"children,omitempty"`
}

// GraphClient abstracts SpiceDB or local implementations
type GraphClient interface {
	Upsert(ctx context.Context, t Tuple) error
	UpsertBatch(ctx context.Context, tuples []Tuple) error
	Check(ctx context.Context, subject RelationRef, relation string, object RelationRef) (bool, string, error)
	Expand(ctx context.Context, relation string, object RelationRef, depth int) (GraphExpansion, error)
}

// CachedGraph wraps a GraphClient with local TTL caching of check results
type CachedGraph struct {
	inner  GraphClient
	ttl    time.Duration
	negTtl time.Duration
	cache  map[string]cacheEntry
}

type cacheEntry struct {
	allow   bool
	source  string
	expires time.Time
}

// NewCachedGraph creates a caching layer
func NewCachedGraph(inner GraphClient, ttl, negTtl time.Duration) *CachedGraph {
	return &CachedGraph{inner: inner, ttl: ttl, negTtl: negTtl, cache: make(map[string]cacheEntry)}
}

func (c *CachedGraph) key(sub RelationRef, rel string, obj RelationRef) string {
	return sub.Namespace + ":" + sub.ObjectID + "#" + rel + "@" + obj.Namespace + ":" + obj.ObjectID
}

func (c *CachedGraph) Upsert(ctx context.Context, t Tuple) error { return c.inner.Upsert(ctx, t) }
func (c *CachedGraph) UpsertBatch(ctx context.Context, tuples []Tuple) error {
	return c.inner.UpsertBatch(ctx, tuples)
}

func (c *CachedGraph) Check(ctx context.Context, subject RelationRef, relation string, object RelationRef) (bool, string, error) {
	k := c.key(subject, relation, object)
	if ent, ok := c.cache[k]; ok && time.Now().Before(ent.expires) {
		return ent.allow, "cache", nil
	}
	allow, source, err := c.inner.Check(ctx, subject, relation, object)
	if err != nil {
		return false, source, err
	}
	ttl := c.ttl
	if !allow && c.negTtl > 0 {
		ttl = c.negTtl
	}
	c.cache[k] = cacheEntry{allow: allow, source: source, expires: time.Now().Add(ttl)}
	return allow, source, nil
}

func (c *CachedGraph) Expand(ctx context.Context, relation string, object RelationRef, depth int) (GraphExpansion, error) {
	return c.inner.Expand(ctx, relation, object, depth)
}

// Clear resets all cache entries (used on write or external invalidation)
func (c *CachedGraph) Clear() { c.cache = make(map[string]cacheEntry) }
