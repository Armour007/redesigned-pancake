package rel

import (
	"context"
	"testing"
	"time"
)

type stubGraph struct {
	calls int
	allow bool
}

func (s *stubGraph) Upsert(ctx context.Context, t Tuple) error             { return nil }
func (s *stubGraph) UpsertBatch(ctx context.Context, tuples []Tuple) error { return nil }
func (s *stubGraph) Check(ctx context.Context, subject RelationRef, relation string, object RelationRef) (bool, string, error) {
	s.calls++
	return s.allow, "stub", nil
}
func (s *stubGraph) Expand(ctx context.Context, relation string, object RelationRef, depth int) (GraphExpansion, error) {
	return GraphExpansion{}, nil
}

func TestCachedGraph_PositiveAndNegativeTTL(t *testing.T) {
	inner := &stubGraph{allow: true}
	cg := NewCachedGraph(inner, 100*time.Millisecond, 50*time.Millisecond)
	sub := RelationRef{"user", "alice"}
	obj := RelationRef{"resource", "r1"}

	// first call hits inner
	if ok, src, _ := cg.Check(context.Background(), sub, "viewer", obj); !ok || src == "cache" {
		t.Fatalf("expected allow via inner, got ok=%v src=%s", ok, src)
	}
	if inner.calls != 1 {
		t.Fatalf("expected 1 call, got %d", inner.calls)
	}
	// second call should be cache
	if ok, src, _ := cg.Check(context.Background(), sub, "viewer", obj); !ok || src != "cache" {
		t.Fatalf("expected cache allow, got ok=%v src=%s", ok, src)
	}
	if inner.calls != 1 {
		t.Fatalf("expected still 1 call, got %d", inner.calls)
	}
	// expire positive TTL
	time.Sleep(120 * time.Millisecond)
	if _, _, _ = cg.Check(context.Background(), sub, "viewer", obj); inner.calls != 2 {
		t.Fatalf("expected inner call after TTL expiry")
	}

	// switch to negative (wait for positive TTL to expire to avoid cached allow)
	inner.allow = false
	inner.calls = 0
	time.Sleep(110 * time.Millisecond)
	if ok, _, _ := cg.Check(context.Background(), sub, "viewer", obj); ok {
		t.Fatalf("expected deny")
	}
	if inner.calls != 1 {
		t.Fatalf("expected 1 call, got %d", inner.calls)
	}
	// cache deny
	if ok, src, _ := cg.Check(context.Background(), sub, "viewer", obj); ok || src != "cache" {
		t.Fatalf("expected cached deny, got ok=%v src=%s", ok, src)
	}
	// wait for negative TTL expire and ensure another inner call
	time.Sleep(60 * time.Millisecond)
	if _, _, _ = cg.Check(context.Background(), sub, "viewer", obj); inner.calls < 2 {
		t.Fatalf("expected another inner call after neg TTL expiry")
	}
}
