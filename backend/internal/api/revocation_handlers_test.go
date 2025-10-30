package api

import (
	"testing"
	"time"
)

func TestMakeETagStable(t *testing.T) {
	t1 := time.Unix(1730200000, 123456789)
	e1 := makeETag("org123", t1)
	e2 := makeETag("org123", t1)
	if e1 != e2 {
		t.Fatalf("etag not stable: %s vs %s", e1, e2)
	}
	e3 := makeETag("org123", t1.Add(time.Nanosecond))
	if e3 == e1 {
		t.Fatalf("etag should change with timestamp change")
	}
}
